package ui

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/grunclepug/vshot/internal/config"
)

type Canvas struct {
	Width, Height int
	ConfigColor   color.RGBA
}

func NewCanvas(width, height int) *Canvas {
	return &Canvas{
		Width:  width,
		Height: height,
		ConfigColor: color.RGBA{
			R: uint8((config.BorderColor >> 16) & 0xFF),
			G: uint8((config.BorderColor >> 8) & 0xFF),
			B: uint8(config.BorderColor & 0xFF),
			A: 255,
		},
	}
}

func (c *Canvas) RenderFrame(state *State, bpp int) []byte {
	frame := image.NewRGBA(image.Rect(0, 0, c.Width, c.Height))
	srcBounds := state.Screenshot.Image.Bounds()
	maskFactor := 1.0 - config.MaskAlpha

	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			gx, gy := srcBounds.Min.X+x, srcBounds.Min.Y+y
			orig := state.Screenshot.Image.RGBAAt(gx, gy)

			if state.HasSelection && gx >= state.Selection.Min.X && gx < state.Selection.Max.X &&
				gy >= state.Selection.Min.Y && gy < state.Selection.Max.Y {
				frame.SetRGBA(x, y, orig)
				continue
			}

			frame.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(orig.R) * maskFactor),
				G: uint8(float64(orig.G) * maskFactor),
				B: uint8(float64(orig.B) * maskFactor),
				A: 255,
			})
		}
	}

	if state.HasSelection {
		sel := state.Selection.Sub(srcBounds.Min)
		c.drawSelectionBorder(frame, sel)
		c.drawGrabHandles(frame, sel)
	}

	return c.toZPixmap(frame, bpp)
}

func (c *Canvas) toZPixmap(img *image.RGBA, bpp int) []byte {
	stride := ((c.Width * bpp / 8) + 3) & ^3
	data := make([]byte, stride*c.Height)

	for y := 0; y < c.Height; y++ {
		srcRow := img.Pix[y*img.Stride : (y+1)*img.Stride]
		dstRow := data[y*stride : (y+1)*stride]

		for x := 0; x < c.Width; x++ {
			i := x * 4
			o := x * (bpp / 8)
			dstRow[o], dstRow[o+1], dstRow[o+2] = srcRow[i+2], srcRow[i+1], srcRow[i]
		}
	}
	return data
}

func (c *Canvas) drawSelectionBorder(frame *image.RGBA, r image.Rectangle) {
	draw.Draw(frame, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1), &image.Uniform{c.ConfigColor}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y), &image.Uniform{c.ConfigColor}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y), &image.Uniform{c.ConfigColor}, image.Point{}, draw.Src)
	draw.Draw(frame, image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y), &image.Uniform{c.ConfigColor}, image.Point{}, draw.Src)
}

func (c *Canvas) drawGrabHandles(frame *image.RGBA, r image.Rectangle) {
	midX, midY := r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2
	for _, pt := range []image.Point{
		{r.Min.X, r.Min.Y},
		{midX, r.Min.Y},
		{r.Max.X - 1, r.Min.Y},
		{r.Max.X - 1, midY},
		{r.Max.X - 1, r.Max.Y - 1},
		{midX, r.Max.Y - 1},
		{r.Min.X, r.Max.Y - 1},
		{r.Min.X, midY},
	} {
		c.drawFilledCircle(frame, pt, config.DotSize)
	}
}

func (c *Canvas) drawFilledCircle(frame *image.RGBA, center image.Point, radius int) {
	r2 := radius * radius
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= r2 {
				frame.SetRGBA(center.X+x, center.Y+y, c.ConfigColor)
			}
		}
	}
}

func CropImage(src *image.RGBA, rect image.Rectangle) *image.RGBA {
	rect = rect.Intersect(src.Bounds())
	if rect.Empty() {
		return image.NewRGBA(image.Rectangle{})
	}

	sub := src.SubImage(rect).(*image.RGBA)
	cropped := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(cropped, cropped.Bounds(), sub, rect.Min, draw.Src)
	return cropped
}
