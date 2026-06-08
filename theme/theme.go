package theme

import (
	raylib "github.com/gen2brain/raylib-go/raylib"
)

var current = map[string]raylib.Color{
	"bg":        raylib.NewColor(30, 30, 30, 255),
	"text":      raylib.NewColor(220, 220, 220, 255),
	"gutter":    raylib.NewColor(100, 100, 100, 255),
	"cursor":    raylib.NewColor(200, 200, 200, 255),
	"selectBg":  raylib.NewColor(80, 120, 180, 255),
	"cursorLn":  raylib.NewColor(45, 45, 45, 255),
	"statusBg":  raylib.NewColor(40, 40, 40, 255),
	"statusTxt": raylib.NewColor(200, 200, 200, 255),
	"searchBg":  raylib.NewColor(255, 200, 0, 180),
	"errorTxt":  raylib.NewColor(255, 100, 100, 255),
}

// Get returns the color for a named theme slot, or white if unknown.
func Get(name string) raylib.Color {
	if c, ok := current[name]; ok {
		return c
	}
	return raylib.NewColor(255, 255, 255, 255)
}

// Set replaces the color for a named theme slot.
func Set(name string, c raylib.Color) {
	current[name] = c
}

// Names returns all configurable theme slot names.
func Names() []string {
	names := make([]string, 0, len(current))
	for n := range current {
		names = append(names, n)
	}
	return names
}
