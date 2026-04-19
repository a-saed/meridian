package renderer_test

import (
	"context"
	"image/color"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/datasource"
	"meridian/internal/renderer"
	"meridian/internal/style"
)

func TestRenderPoint(t *testing.T) {
	r := renderer.NewVectorRenderer()

	req := renderer.Request{
		Features: []datasource.Feature{
			{Geometry: orb.Point{0.5, 0.5}},
		},
		Style: style.Style{
			FillColor:   "#ff0000",
			StrokeColor: "#000000",
			StrokeWidth: 1.0,
			Opacity:     1.0,
		},
		Width:  256,
		Height: 256,
		Bbox:   [4]float64{0, 0, 1, 1},
	}

	img, err := r.Render(context.Background(), req)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 256 {
		t.Errorf("want 256x256, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// The center pixel should not be fully transparent (point was drawn there)
	cx, cy := 128, 128
	_, _, _, a := img.At(cx, cy).RGBA()
	if a == 0 {
		t.Error("expected non-transparent pixel at center where point was drawn")
	}
}

func TestRenderPolygon(t *testing.T) {
	r := renderer.NewVectorRenderer()

	req := renderer.Request{
		Features: []datasource.Feature{
			{
				Geometry: orb.Polygon{
					{{0.1, 0.1}, {0.9, 0.1}, {0.9, 0.9}, {0.1, 0.9}, {0.1, 0.1}},
				},
			},
		},
		Style:  style.Default(),
		Width:  256,
		Height: 256,
		Bbox:   [4]float64{0, 0, 1, 1},
	}

	img, err := r.Render(context.Background(), req)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	_ = img // verify it renders without panic
}

func TestRenderEmpty(t *testing.T) {
	r := renderer.NewVectorRenderer()
	req := renderer.Request{
		Features: nil,
		Style:    style.Default(),
		Width:    256,
		Height:   256,
		Bbox:     [4]float64{0, 0, 1, 1},
	}
	img, err := r.Render(context.Background(), req)
	if err != nil {
		t.Fatalf("render empty: %v", err)
	}
	// Should be a fully transparent image
	_, _, _, a := img.At(128, 128).RGBA()
	if a != 0 {
		t.Errorf("expected transparent image for empty features, got alpha=%d", a)
	}
}

func TestPoolRenderer(t *testing.T) {
	base := renderer.NewVectorRenderer()
	r := renderer.WithPool(base)

	req := renderer.Request{
		Features: []datasource.Feature{{Geometry: orb.Point{0.5, 0.5}}},
		Style:    style.Default(),
		Width:    64,
		Height:   64,
		Bbox:     [4]float64{0, 0, 1, 1},
	}

	img, err := r.Render(context.Background(), req)
	if err != nil {
		t.Fatalf("pool render: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 64 {
		t.Errorf("want 64 wide, got %d", bounds.Dx())
	}

	_ = color.NRGBA{}
}
