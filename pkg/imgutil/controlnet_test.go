package imgutil_test

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"aris/pkg/imgutil"
)

func createSyntheticTestImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Black background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)
	// White square in the middle [w/4, h/4] to [3w/4, 3h/4]
	for y := h / 4; y < 3*h/4; y++ {
		for x := w / 4; x < 3*w/4; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	return img
}

func TestCannyEdgeDetection_Synthetic(t *testing.T) {
	w, h := 100, 100
	synthetic := createSyntheticTestImage(w, h)

	edges, err := imgutil.CannyEdgeDetection(synthetic, 50, 150)
	if err != nil {
		t.Fatalf("unexpected error during Canny edge detection: %v", err)
	}

	if edges == nil {
		t.Fatal("expected edge image, got nil")
	}

	bounds := edges.Bounds()
	if bounds.Dx() != w || bounds.Dy() != h {
		t.Fatalf("expected output bounds %dx%d, got %dx%d", w, h, bounds.Dx(), bounds.Dy())
	}

	// Center of square (inside) should be black (no edge)
	centerColor := edges.At(50, 50)
	r, g, b, _ := centerColor.RGBA()
	if r > 0 || g > 0 || b > 0 {
		t.Errorf("center of solid shape should be suppressed (0,0,0), got r=%d g=%d b=%d", r, g, b)
	}

	// Boundary pixels (e.g. at x=25, y=50) should detect an edge
	edgeCount := 0
	for y := 20; y <= 30; y++ {
		r, _, _, _ := edges.At(25, y).RGBA()
		if r > 0 {
			edgeCount++
		}
	}
	if edgeCount == 0 {
		t.Errorf("expected to find strong edge pixels around boundary x=25")
	}
}

func TestPreprocessControlNet_Types(t *testing.T) {
	synthetic := createSyntheticTestImage(64, 64)

	// Canny type -> runs canny
	cannyOut, err := imgutil.PreprocessControlNet("canny", synthetic)
	if err != nil {
		t.Fatalf("unexpected error preprocessing canny: %v", err)
	}
	if cannyOut == nil {
		t.Fatal("expected canny result, got nil")
	}

	// Other types (depth, openpose) -> pass-through unchanged
	passthrough, err := imgutil.PreprocessControlNet("depth", synthetic)
	if err != nil {
		t.Fatalf("unexpected error preprocessing depth: %v", err)
	}
	if passthrough != synthetic {
		t.Errorf("expected passthrough image to match original pointer/image")
	}
}

func TestPreprocessCannyFile_E2E(t *testing.T) {
	tempDir := t.TempDir()
	inPath := filepath.Join(tempDir, "input.png")
	outPath := filepath.Join(tempDir, "output_canny.png")

	synthetic := createSyntheticTestImage(80, 80)
	outFile, err := os.Create(inPath)
	if err != nil {
		t.Fatalf("failed to create temp test image: %v", err)
	}
	if err := imgutil.SavePNG(outFile, synthetic); err != nil {
		outFile.Close()
		t.Fatalf("failed to save temp PNG: %v", err)
	}
	outFile.Close()

	// Preprocess file
	err = imgutil.PreprocessCannyFile(inPath, outPath, 100, 200)
	if err != nil {
		t.Fatalf("PreprocessCannyFile failed: %v", err)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("expected output edge file %s to exist", outPath)
	}
}
