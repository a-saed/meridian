package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var Static embed.FS

// StaticHandler serves everything under static/ stripped of the "static/" prefix.
func StaticHandler() http.Handler {
	sub, _ := fs.Sub(Static, "static")
	return http.FileServer(http.FS(sub))
}
