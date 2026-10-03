package wayland

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// shm formats vshot can read and write. Both are 32-bit little endian.
const (
	formatARGB8888 uint32 = 0          // bytes B, G, R, A
	formatXRGB8888 uint32 = 1          // bytes B, G, R, X
	formatABGR8888 uint32 = 0x34324241 // bytes R, G, B, A
	formatXBGR8888 uint32 = 0x34324258 // bytes R, G, B, X
)

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
