package wayland

import (
	"fmt"
	"image"
	"strings"
)

// captureOutput copies the current contents of o using wlr-screencopy.
func (s *session) captureOutput(o *output) (*image.RGBA, error) {
	if o.transform != 0 {
		return nil, fmt.Errorf("output %s: rotated or flipped outputs are not supported", o.label())
	}

	var (
		format                uint32
		pix                   layout
		width, height, stride int
		offered               []string // every shm format the compositor proposed
		haveFormat            bool
		bufferDone, yInvert   bool
		ready, failed         bool
	)
	c := s.c
	frame := c.newID(func(op uint16, m *msg) {
		switch op {
		case 0: // buffer
			f, w, h, st := m.uint(), m.uint(), m.uint(), m.uint()
			offered = append(offered, formatName(f))
			if l, ok := layouts[f]; ok && !haveFormat {
				format, pix, width, height, stride, haveFormat = f, l, int(w), int(h), int(st), true
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
	for !failed && !bufferDone && !(len(offered) > 0 && s.screencopyVersion < 3) {
		if err := c.dispatch(); err != nil {
			return nil, err
		}
	}
	if failed {
		return nil, fmt.Errorf("output %s: capture failed", o.label())
	}
	if !haveFormat {
		return nil, fmt.Errorf("output %s: no supported shm capture format (compositor offered: %s)",
			o.label(), strings.Join(offered, ", "))
	}
	if width <= 0 || height <= 0 || stride < width*pix.size {
		return nil, fmt.Errorf("output %s: bad capture size %dx%d, stride %d", o.label(), width, height, stride)
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
	for y := range height {
		srcY := y
		if yInvert {
			srcY = height - 1 - y
		}
		pix.decodeRow(img.Pix[y*img.Stride:][:width*4], buf.data[srcY*stride:][:width*pix.size])
	}
	return img, nil
}
