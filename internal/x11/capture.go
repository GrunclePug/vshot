package x11

import (
	"fmt"
	"image"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
)

// Capture grabs the current screen contents. Unless all is set, only the
// monitor under the pointer is captured. It returns the image (origin 0,0)
// and the area it covers in root window coordinates.
func Capture(X *xgb.Conn, all bool) (*image.RGBA, image.Rectangle, error) {
	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)
	if err := checkFormat(setup, screen); err != nil {
		return nil, image.Rectangle{}, err
	}

	bounds := image.Rect(0, 0, int(screen.WidthInPixels), int(screen.HeightInPixels))
	if !all {
		if m, ok := monitorAtPointer(X, screen.Root); ok {
			bounds = m.Intersect(bounds)
		}
	}

	reply, err := xproto.GetImage(X, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
		int16(bounds.Min.X), int16(bounds.Min.Y), uint16(bounds.Dx()), uint16(bounds.Dy()),
		0xffffffff).Reply()
	if err != nil {
		return nil, image.Rectangle{}, err
	}

	img := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if len(reply.Data) < len(img.Pix) {
		return nil, image.Rectangle{}, fmt.Errorf("short image reply from X server")
	}
	// ZPixmap at 32bpp, LSB first: B, G, R, X.
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] =
			reply.Data[i+2], reply.Data[i+1], reply.Data[i], 0xff
	}
	return img, bounds, nil
}

// checkFormat rejects pixel layouts other than 8-bit-per-channel TrueColor
// packed into 32 bits, which is what Capture and UploadImage assume.
func checkFormat(setup *xproto.SetupInfo, screen *xproto.ScreenInfo) error {
	if setup.ImageByteOrder != xproto.ImageOrderLSBFirst {
		return fmt.Errorf("unsupported X server: MSB-first image byte order")
	}
	if screen.RootDepth != 24 && screen.RootDepth != 32 {
		return fmt.Errorf("unsupported root depth %d (need 24 or 32)", screen.RootDepth)
	}
	for _, f := range setup.PixmapFormats {
		if f.Depth == screen.RootDepth && f.BitsPerPixel != 32 {
			return fmt.Errorf("unsupported pixmap format: %d bits per pixel", f.BitsPerPixel)
		}
	}
	return nil
}

func monitorAtPointer(X *xgb.Conn, root xproto.Window) (image.Rectangle, bool) {
	if err := xinerama.Init(X); err != nil {
		return image.Rectangle{}, false
	}
	screens, err := xinerama.QueryScreens(X).Reply()
	if err != nil {
		return image.Rectangle{}, false
	}
	ptr, err := xproto.QueryPointer(X, root).Reply()
	if err != nil {
		return image.Rectangle{}, false
	}

	p := image.Pt(int(ptr.RootX), int(ptr.RootY))
	for _, s := range screens.ScreenInfo {
		r := image.Rect(int(s.XOrg), int(s.YOrg), int(s.XOrg)+int(s.Width), int(s.YOrg)+int(s.Height))
		if p.In(r) {
			return r, true
		}
	}
	return image.Rectangle{}, false
}
