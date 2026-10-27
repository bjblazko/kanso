// SPDX-License-Identifier: Apache-2.0

// Command gen-icons draws Kanso's app icon: a dark square with three light
// note lines and one orange dot (the accent). It writes PNGs for the web app
// manifest and iOS. Run from the repository root: go run ./scripts/gen-icons
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var (
	dark   = color.NRGBA{0x1E, 0x1F, 0x21, 0xFF} // --fg (light theme)
	light  = color.NRGBA{0xF4, 0xF4, 0xF0, 0xFF} // --bg (light theme)
	accent = color.NRGBA{0xE8, 0x5D, 0x04, 0xFF} // --accent
)

const supersample = 4

// shape reports the colour at (x, y) in a 1×1 square, or false for the background.
type shape func(x, y float64) (color.NRGBA, bool)

// bar is a rounded horizontal line from x0 to x1 at height y with radius r.
func bar(x0, x1, y, r float64, c color.NRGBA) shape {
	return func(x, py float64) (color.NRGBA, bool) {
		cx := math.Max(x0, math.Min(x1, x))
		return c, math.Hypot(x-cx, py-y) <= r
	}
}

func dot(cx, cy, r float64, c color.NRGBA) shape {
	return func(x, y float64) (color.NRGBA, bool) { return c, math.Hypot(x-cx, y-cy) <= r }
}

// The mark stays inside the central 60 %, the safe zone of maskable icons.
var mark = []shape{
	bar(0.30, 0.70, 0.36, 0.035, light),
	bar(0.30, 0.70, 0.50, 0.035, light),
	bar(0.30, 0.54, 0.64, 0.035, light),
	dot(0.665, 0.64, 0.05, accent),
}

func colourAt(x, y float64) color.NRGBA {
	for _, s := range mark {
		if c, ok := s(x, y); ok {
			return c
		}
	}
	return dark
}

// draw renders the icon at size×size pixels with supersampled edges.
func draw(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	n := float64(size * supersample)
	for py := range size {
		for px := range size {
			var r, g, b float64
			for sy := range supersample {
				for sx := range supersample {
					c := colourAt((float64(px*supersample+sx)+0.5)/n, (float64(py*supersample+sy)+0.5)/n)
					r, g, b = r+float64(c.R), g+float64(c.G), b+float64(c.B)
				}
			}
			k := float64(supersample * supersample)
			img.SetNRGBA(px, py, color.NRGBA{uint8(r / k), uint8(g / k), uint8(b / k), 0xFF})
		}
	}
	return img
}

func write(dir, name string, size int) error {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if err := png.Encode(f, draw(size)); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func main() {
	dir := filepath.Join("web", "static", "icons")
	sizes := map[string]int{"icon-192.png": 192, "icon-512.png": 512, "apple-touch-icon.png": 180}
	for name, size := range sizes {
		if err := write(dir, name, size); err != nil {
			fmt.Fprintln(os.Stderr, "gen-icons:", err)
			os.Exit(1)
		}
	}
}
