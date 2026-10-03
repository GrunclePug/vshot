package wayland

/*
#cgo pkg-config: xkbcommon
#include <stdlib.h>
#include <xkbcommon/xkbcommon.h>
*/
import "C"

import (
	"bytes"
	"errors"
	"unsafe"

	"golang.org/x/sys/unix"
)

// keymap translates evdev key codes into keysyms using the keymap and
// modifier state sent by the compositor.
type keymap struct {
	ctx    *C.struct_xkb_context
	keymap *C.struct_xkb_keymap
	state  *C.struct_xkb_state
}

func newKeymap() (*keymap, error) {
	ctx := C.xkb_context_new(C.XKB_CONTEXT_NO_FLAGS)
	if ctx == nil {
		return nil, errors.New("xkb_context_new failed")
	}
	return &keymap{ctx: ctx}, nil
}

// load replaces the keymap with the XKB text keymap in fd.
func (k *keymap) load(fd int, size uint32) error {
	defer unix.Close(fd)
	data, err := unix.Mmap(fd, 0, int(size), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		return err
	}
	defer unix.Munmap(data)
	if n := bytes.IndexByte(data, 0); n >= 0 {
		data = data[:n]
	}
	if len(data) == 0 {
		return errors.New("empty keymap")
	}

	km := C.xkb_keymap_new_from_buffer(k.ctx, (*C.char)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		C.XKB_KEYMAP_FORMAT_TEXT_V1, C.XKB_KEYMAP_COMPILE_NO_FLAGS)
	if km == nil {
		return errors.New("could not compile keymap")
	}
	st := C.xkb_state_new(km)
	if st == nil {
		C.xkb_keymap_unref(km)
		return errors.New("xkb_state_new failed")
	}
	k.free()
	k.keymap, k.state = km, st
	return nil
}

func (k *keymap) setModifiers(depressed, latched, locked, group uint32) {
	if k.state != nil {
		C.xkb_state_update_mask(k.state, C.xkb_mod_mask_t(depressed), C.xkb_mod_mask_t(latched),
			C.xkb_mod_mask_t(locked), 0, 0, C.xkb_layout_index_t(group))
	}
}

// keysym returns the keysym for an evdev key code, or 0 without a keymap.
func (k *keymap) keysym(key uint32) uint32 {
	if k.state == nil {
		return 0
	}
	return uint32(C.xkb_state_key_get_one_sym(k.state, C.xkb_keycode_t(key+8)))
}

func (k *keymap) free() {
	if k.state != nil {
		C.xkb_state_unref(k.state)
	}
	if k.keymap != nil {
		C.xkb_keymap_unref(k.keymap)
	}
	k.keymap, k.state = nil, nil
}
