package wayland

import (
	"fmt"
	"image"
)

// captureOutput copies the current contents of o using wlr-screencopy.
func (s *session) captureOutput(o *output) (*image.RGBA, error) {
	if o.transform != 0 {
		return nil, fmt.Errorf("output %s: rotated or flipped outputs are not supported", o.label())
	}

	var (
		format                          uint32
		width, height, stride           int
		haveFormat, bufferDone, yInvert bool
		ready, failed                   bool
	)
	c := s.c
	frame := c.newID(func(op uint16, m *msg) {
		switch op {
		case 0: // buffer
			f, w, h, st := m.uint(), m.uint(), m.uint(), m.uint()
			if !haveFormat && rgbOrder(f) >= 0 {
				format, width, height, stride, haveFormat = f, int(w), int(h), int(st), true
			}
		case 1: // flags
			yInvert = m.uint()&1 != 0
		case 2: // ready
			ready = true
		case 3: // failed
			failed = true
		case 6: // buffer_done
			bufferDone = true
		}
	})
	if err := c.send(s.screencopy, 0, frame, int32(0), o.id); err != nil { // capture_output
		return nil, err
	}
	defer c.send(frame, 1) // destroy

	// Version 3 lists every offered format, then sends buffer_done. Older
	// versions offer a single shm buffer.
	for !failed && !bufferDone && !(haveFormat && s.screencopyVersion < 3) {
		if err := c.dispatch(); err != nil {
			return nil, err
		}
	}
	if failed {
		return nil, fmt.Errorf("output %s: capture failed", o.label())
	}
	if !haveFormat {
		return nil, fmt.Errorf("output %s: no supported shm capture format", o.label())
	}

	buf, err := newShmBuffer(c, s.shm, width, height, stride, format)
	if err != nil {
		return nil, err
	}
	defer buf.destroy()
	if err := c.send(frame, 0, buf.id); err != nil { // copy
		return nil, err
	}
	for !ready && !failed {
		if err := c.dispatch(); err != nil {
			return nil, err
		}
	}
	if failed {
		return nil, fmt.Errorf("output %s: capture failed", o.label())
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	swap := rgbOrder(format) == 0
	for y := range height {
		srcY := y
		if yInvert {
			srcY = height - 1 - y
		}
		src := buf.data[srcY*stride : srcY*stride+width*4]
		dst := img.Pix[y*img.Stride : y*img.Stride+width*4]
		for i := 0; i < len(src); i += 4 {
			if swap {
				dst[i], dst[i+1], dst[i+2] = src[i+2], src[i+1], src[i]
			} else {
				dst[i], dst[i+1], dst[i+2] = src[i], src[i+1], src[i+2]
			}
			dst[i+3] = 0xff
		}
	}
	return img, nil
}

// rgbOrder reports the byte order of a supported shm format: 0 for
// B,G,R,x, 1 for R,G,B,x, or -1 if the format is not supported.
func rgbOrder(format uint32) int {
	switch format {
	case formatARGB8888, formatXRGB8888:
		return 0
	case formatABGR8888, formatXBGR8888:
		return 1
	}
	return -1
}
