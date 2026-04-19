package style

// Style defines how features are rendered on a map.
type Style struct {
	ID          string
	Name        string
	FillColor   string  // hex e.g. "#3388ff"
	StrokeColor string  // hex e.g. "#ffffff"
	StrokeWidth float64 // pixels
	Opacity     float64 // 0.0–1.0
}

// Default returns a sensible out-of-the-box style.
func Default() Style {
	return Style{
		FillColor:   "#3388ff",
		StrokeColor: "#ffffff",
		StrokeWidth: 1.0,
		Opacity:     1.0,
	}
}
