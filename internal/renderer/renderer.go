package renderer

import (
	"context"
	"image"

	"meridian/internal/datasource"
	"meridian/internal/style"
)

// Request holds everything the renderer needs to produce a map image.
type Request struct {
	Features []datasource.Feature
	Style    style.Style
	Width    int
	Height   int
	Bbox     [4]float64 // [minX, minY, maxX, maxY] in the render CRS
}

// Renderer converts a set of features into a raster image.
type Renderer interface {
	Render(ctx context.Context, req Request) (image.Image, error)
}
