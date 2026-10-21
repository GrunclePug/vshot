package wayland

import "testing"

func TestDecodeRow(t *testing.T) {
	want := []byte{0x11, 0x22, 0x33, 0xff}
	// 10-bit channels whose top 8 bits are 0x11, 0x22 and 0x33.
	const r10, g10, b10 = 0x11<<2 | 1, 0x22<<2 | 2, 0x33<<2 | 3
	le := func(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} }

	for _, tc := range []struct {
		format uint32
		src    []byte
	}{
		{formatXRGB8888, []byte{0x33, 0x22, 0x11, 0x00}},
		{formatARGB8888, []byte{0x33, 0x22, 0x11, 0x80}},
		{formatXBGR8888, []byte{0x11, 0x22, 0x33, 0x00}},
		{formatABGR8888, []byte{0x11, 0x22, 0x33, 0x80}},
		{formatRGBX8888, []byte{0x00, 0x33, 0x22, 0x11}},
		{formatBGRA8888, []byte{0x80, 0x11, 0x22, 0x33}},
		{formatRGB888, []byte{0x33, 0x22, 0x11}},
		{formatBGR888, []byte{0x11, 0x22, 0x33}},
		{formatXRGB2101010, le(r10<<20 | g10<<10 | b10)},
		{formatABGR2101010, le(3<<30 | b10<<20 | g10<<10 | r10)},
	} {
		l, ok := layouts[tc.format]
		if !ok || len(tc.src) != l.size {
			t.Errorf("%s: layout %+v, ok=%v for a %d-byte pixel", formatName(tc.format), l, ok, len(tc.src))
			continue
		}
		// Two pixels, so a wrong pixel size shows up in the second.
		got := make([]byte, 8)
		l.decodeRow(got, append(append([]byte{}, tc.src...), tc.src...))
		if string(got[:4]) != string(want) || string(got[4:]) != string(want) {
			t.Errorf("%s: decoded % x, want % x twice", formatName(tc.format), got, want)
		}
	}
}

func TestFormatName(t *testing.T) {
	for format, want := range map[uint32]string{
		formatXRGB8888: "XR24",
		formatBGR888:   "BG24",
		0x00000102:     "0x102",
	} {
		if got := formatName(format); got != want {
			t.Errorf("formatName(%#x) = %q, want %q", format, got, want)
		}
	}
}
