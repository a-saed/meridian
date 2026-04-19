package style_test

import (
	"testing"

	"meridian/internal/style"
)

func TestDefault(t *testing.T) {
	s := style.Default()
	if s.FillColor != "#3388ff" {
		t.Errorf("want FillColor #3388ff, got %s", s.FillColor)
	}
	if s.Opacity != 1.0 {
		t.Errorf("want Opacity 1.0, got %f", s.Opacity)
	}
	if s.StrokeWidth != 1.0 {
		t.Errorf("want StrokeWidth 1.0, got %f", s.StrokeWidth)
	}
}
