package display

import (
	"image"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
)

type Screenshot struct {
	Image  *image.RGBA
	Bounds image.Rectangle
}

type MonitorInfo struct{ X, Y, Width, Height int }

func CaptureScreen(X *xgb.Conn, allDisplays bool) (*Screenshot, error) {
	screen := xproto.Setup(X).DefaultScreen(X)

	// Determine capture area
	monitors, _ := getMonitors(X)
	target := image.Rect(0, 0, int(screen.WidthInPixels), int(screen.HeightInPixels))

	if !allDisplays && len(monitors) > 0 {
		ptr, err := xproto.QueryPointer(X, screen.Root).Reply()
		if err == nil {
			for _, m := range monitors {
				if ptr.RootX >= int16(m.X) && ptr.RootX < int16(m.X+m.Width) &&
					ptr.RootY >= int16(m.Y) && ptr.RootY < int16(m.Y+m.Height) {
					target = image.Rect(m.X, m.Y, m.X+m.Width, m.Y+m.Height)
					break
				}
			}
		}
	}

	// Capture pixels
	reply, err := xproto.GetImage(X, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
		int16(target.Min.X), int16(target.Min.Y), uint16(target.Dx()), uint16(target.Dy()), 0xffffffff).Reply()
	if err != nil {
		return nil, err
	}

	// Transform BGRA to RGBA efficiently
	img := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	for i := 0; i < len(reply.Data); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = reply.Data[i+2], reply.Data[i+1], reply.Data[i], 255
	}

	return &Screenshot{Image: img, Bounds: target}, nil
}

func getMonitors(X *xgb.Conn) ([]MonitorInfo, error) {
	if err := xinerama.Init(X); err != nil {
		return nil, nil
	}
	reply, err := xinerama.QueryScreens(X).Reply()
	if err != nil {
		return nil, err
	}

	monitors := make([]MonitorInfo, len(reply.ScreenInfo))
	for i, s := range reply.ScreenInfo {
		monitors[i] = MonitorInfo{int(s.XOrg), int(s.YOrg), int(s.Width), int(s.Height)}
	}
	return monitors, nil
}
