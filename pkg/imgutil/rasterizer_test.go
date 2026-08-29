package imgutil_test

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"aris/pkg/imgutil"
)

func TestRenderImageANSI(t *testing.T) {
	// Create a small 4x4 test image (red on top, blue on bottom)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	for y := 2; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}

	ansiStr := imgutil.RenderImageANSI(img, 10, 10)
	if ansiStr == "" {
		t.Fatalf("expected non-empty ANSI output")
	}

	// Verify ANSI escapes and halfblock character
	if !strings.Contains(ansiStr, "▀") {
		t.Errorf("expected halfblock character in output")
	}
	if !strings.Contains(ansiStr, "\x1b[38;2;") {
		t.Errorf("expected 24-bit foreground color escape code")
	}
}
