package renderer

import (
	"context"
	"image"
	"runtime"
)

type poolRenderer struct {
	inner Renderer
	sem   chan struct{}
}

// WithPool wraps r in a bounded semaphore that limits concurrent renders
// to GOMAXPROCS*2. This prevents memory spikes under high concurrency.
func WithPool(r Renderer) Renderer {
	size := runtime.GOMAXPROCS(0) * 2
	if size < 2 {
		size = 2
	}
	return &poolRenderer{inner: r, sem: make(chan struct{}, size)}
}

func (p *poolRenderer) Render(ctx context.Context, req Request) (image.Image, error) {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
		return p.inner.Render(ctx, req)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
