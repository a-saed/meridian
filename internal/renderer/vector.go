package renderer

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"github.com/paulmach/orb"

	"meridian/internal/style"
)

type vectorRenderer struct{}

// NewVectorRenderer returns a Renderer that draws vector features using gg.
func NewVectorRenderer() Renderer {
	return &vectorRenderer{}
}

func (v *vectorRenderer) Render(_ context.Context, req Request) (image.Image, error) {
	dc := gg.NewContext(req.Width, req.Height)
	dc.SetColor(color.Transparent)
	dc.Clear()

	if len(req.Features) == 0 {
		return dc.Image(), nil
	}

	dx := req.Bbox[2] - req.Bbox[0]
	dy := req.Bbox[3] - req.Bbox[1]
	if dx == 0 || dy == 0 {
		return nil, fmt.Errorf("renderer: degenerate bbox")
	}

	scaleX := float64(req.Width) / dx
	scaleY := float64(req.Height) / dy

	toPixel := func(x, y float64) (float64, float64) {
		px := (x - req.Bbox[0]) * scaleX
		py := float64(req.Height) - (y-req.Bbox[1])*scaleY // flip Y axis
		return px, py
	}

	for _, f := range req.Features {
		drawGeometry(dc, f.Geometry, req.Style, toPixel)
	}
	return dc.Image(), nil
}

func drawGeometry(dc *gg.Context, g orb.Geometry, s style.Style, toPixel func(float64, float64) (float64, float64)) {
	switch geom := g.(type) {
	case orb.Point:
		px, py := toPixel(geom[0], geom[1])
		dc.DrawCircle(px, py, 6)
		applyFill(dc, s)
		dc.FillPreserve()
		applyStroke(dc, s)
		dc.Stroke()

	case orb.LineString:
		if len(geom) == 0 {
			return
		}
		for i, p := range geom {
			px, py := toPixel(p[0], p[1])
			if i == 0 {
				dc.MoveTo(px, py)
			} else {
				dc.LineTo(px, py)
			}
		}
		applyStroke(dc, s)
		dc.Stroke()

	case orb.Polygon:
		for _, ring := range geom {
			if len(ring) == 0 {
				continue
			}
			for i, p := range ring {
				px, py := toPixel(p[0], p[1])
				if i == 0 {
					dc.MoveTo(px, py)
				} else {
					dc.LineTo(px, py)
				}
			}
			dc.ClosePath()
		}
		applyFill(dc, s)
		dc.FillPreserve()
		applyStroke(dc, s)
		dc.Stroke()

	case orb.MultiPoint:
		for _, pt := range geom {
			drawGeometry(dc, pt, s, toPixel)
		}
	case orb.MultiLineString:
		for _, ls := range geom {
			drawGeometry(dc, ls, s, toPixel)
		}
	case orb.MultiPolygon:
		for _, poly := range geom {
			drawGeometry(dc, poly, s, toPixel)
		}
	case orb.Collection:
		for _, child := range geom {
			drawGeometry(dc, child, s, toPixel)
		}
	}
}

func applyFill(dc *gg.Context, s style.Style) {
	c := parseHex(s.FillColor)
	c.A = uint8(s.Opacity * 255)
	dc.SetColor(c)
}

func applyStroke(dc *gg.Context, s style.Style) {
	dc.SetLineWidth(s.StrokeWidth)
	dc.SetColor(parseHex(s.StrokeColor))
}

// parseHex parses a "#rrggbb" or "#rgb" color string.
func parseHex(hex string) color.NRGBA {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	switch len(hex) {
	case 6:
		fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	case 3:
		fmt.Sscanf(hex, "%1x%1x%1x", &r, &g, &b)
		r, g, b = r*17, g*17, b*17
	default:
		return color.NRGBA{R: 51, G: 136, B: 255, A: 255}
	}
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

var _ Renderer = (*vectorRenderer)(nil)
