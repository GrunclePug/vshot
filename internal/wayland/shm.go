package wayland

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

// shm formats vshot understands. Overlays are always drawn as XRGB8888;
// the rest are capture formats, which the compositor picks. Mesa drivers
// hand out the 32-bit ones, NVIDIA's driver 24-bit BGR888.
const (
	formatARGB8888    uint32 = 0          // bytes B, G, R, A
	formatXRGB8888    uint32 = 1          // bytes B, G, R, X
	formatABGR8888    uint32 = 0x34324241 // bytes R, G, B, A
	formatXBGR8888    uint32 = 0x34324258 // bytes R, G, B, X
	formatRGBA8888    uint32 = 0x34324152 // bytes A, B, G, R
	formatRGBX8888    uint32 = 0x34325852 // bytes X, B, G, R
	formatBGRA8888    uint32 = 0x34324142 // bytes A, R, G, B
	formatBGRX8888    uint32 = 0x34325842 // bytes X, R, G, B
	formatRGB888      uint32 = 0x34324752 // bytes B, G, R
	formatBGR888      uint32 = 0x34324742 // bytes R, G, B
	formatARGB2101010 uint32 = 0x30335241 // 32-bit word A:R:G:B, 2:10:10:10
	formatXRGB2101010 uint32 = 0x30335258
	formatABGR2101010 uint32 = 0x30334241 // 32-bit word A:B:G:R, 2:10:10:10
	formatXBGR2101010 uint32 = 0x30334258
)

// layout says where an shm format keeps its color channels: as byte offsets
// within the pixel, or with deep set as bit offsets of 10-bit channels
// within a little-endian 32-bit word.
type layout struct {
	size    int // bytes per pixel
	r, g, b int
	deep    bool
}

var layouts = map[uint32]layout{
	formatARGB8888:    {size: 4, r: 2, g: 1, b: 0},
	formatXRGB8888:    {size: 4, r: 2, g: 1, b: 0},
	formatABGR8888:    {size: 4, r: 0, g: 1, b: 2},
	formatXBGR8888:    {size: 4, r: 0, g: 1, b: 2},
	formatRGBA8888:    {size: 4, r: 3, g: 2, b: 1},
	formatRGBX8888:    {size: 4, r: 3, g: 2, b: 1},
	formatBGRA8888:    {size: 4, r: 1, g: 2, b: 3},
	formatBGRX8888:    {size: 4, r: 1, g: 2, b: 3},
	formatRGB888:      {size: 3, r: 2, g: 1, b: 0},
	formatBGR888:      {size: 3, r: 0, g: 1, b: 2},
	formatARGB2101010: {size: 4, r: 20, g: 10, b: 0, deep: true},
	formatXRGB2101010: {size: 4, r: 20, g: 10, b: 0, deep: true},
	formatABGR2101010: {size: 4, r: 0, g: 10, b: 20, deep: true},
	formatXBGR2101010: {size: 4, r: 0, g: 10, b: 20, deep: true},
}

// decodeRow converts a row of pixels from src into opaque R, G, B, A bytes
// in dst, which sets the number of pixels.
func (l layout) decodeRow(dst, src []byte) {
	for i, j := 0, 0; i+4 <= len(dst); i, j = i+4, j+l.size {
		if l.deep {
			// Keep the top 8 of each channel's 10 bits.
			v := binary.LittleEndian.Uint32(src[j:])
			dst[i], dst[i+1], dst[i+2] = uint8(v>>(l.r+2)), uint8(v>>(l.g+2)), uint8(v>>(l.b+2))
		} else {
			dst[i], dst[i+1], dst[i+2] = src[j+l.r], src[j+l.g], src[j+l.b]
		}
		dst[i+3] = 0xff
	}
}

// formatName returns the fourcc code of an shm format, for error messages.
func formatName(format uint32) string {
	switch format {
	case formatARGB8888:
		return "AR24"
	case formatXRGB8888:
		return "XR24"
	}
	b := binary.LittleEndian.AppendUint32(nil, format)
	for _, ch := range b {
		if ch < 0x20 || ch > 0x7e {
			return fmt.Sprintf("%#x", format)
		}
	}
	return string(b)
}

// shmBuffer is a wl_buffer backed by a shared memory mapping.
type shmBuffer struct {
	c                     *conn
	id                    uint32
	data                  []byte
	width, height, stride int
	busy                  bool // attached and not yet released by the compositor
}

func newShmBuffer(c *conn, shm uint32, width, height, stride int, format uint32) (*shmBuffer, error) {
	size := stride * height
	f, err := unix.MemfdCreate("vshot", unix.MFD_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("memfd_create: %w", err)
	}
	defer unix.Close(f)
	if err := unix.Ftruncate(f, int64(size)); err != nil {
		return nil, err
	}
	data, err := unix.Mmap(f, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}

	b := &shmBuffer{c: c, data: data, width: width, height: height, stride: stride}
	pool := c.newID(nil)
	b.id = c.newID(func(op uint16, _ *msg) {
		if op == 0 { // release
			b.busy = false
		}
	})
	// The pool is only needed to create the buffer; the buffer keeps the
	// memory alive on the compositor side.
	if err := c.send(shm, 0, pool, fd(f), int32(size)); err != nil { // create_pool
		return nil, err
	}
	if err := c.send(pool, 0, b.id, int32(0), int32(width), int32(height), int32(stride), format); err != nil { // create_buffer
		return nil, err
	}
	return b, c.send(pool, 1) // destroy
}

func (b *shmBuffer) destroy() {
	b.c.send(b.id, 0)
	unix.Munmap(b.data)
}
