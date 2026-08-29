package imgutil_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aris/pkg/imgutil"
)

func createTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func createTestJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func TestLoadAndValidateImage_LocalValid(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")
	pngBytes := createTestPNG(128, 128)
	if err := os.WriteFile(pngPath, pngBytes, 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}

	data, mime, err := imgutil.LoadAndValidateImage(pngPath, imgutil.MaxImageSize)
	if err != nil {
		t.Fatalf("expected valid load, got error: %v", err)
	}
	if len(data) != len(pngBytes) {
		t.Errorf("expected %d bytes, got %d", len(pngBytes), len(data))
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got %s", mime)
	}
}

func TestLoadAndValidateImage_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	jpgPath := filepath.Join(tmpDir, "test.jpg")
	jpgBytes := createTestJPEG(64, 64)
	if err := os.WriteFile(jpgPath, jpgBytes, 0644); err != nil {
		t.Fatalf("failed to write test jpg: %v", err)
	}

	data, mime, err := imgutil.LoadAndValidateImage(jpgPath, 0)
	if err != nil {
		t.Fatalf("expected valid load, got error: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("expected non-empty data")
	}
	if mime != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", mime)
	}
}

func TestLoadAndValidateImage_NonExistent(t *testing.T) {
	_, _, err := imgutil.LoadAndValidateImage("/path/to/non_existent_file.png", 0)
	if err == nil {
		t.Fatalf("expected error for non-existent file, got nil")
	}
}

func TestLoadAndValidateImage_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, []byte("this is not an image"), 0644); err != nil {
		t.Fatalf("failed to write text file: %v", err)
	}

	_, _, err := imgutil.LoadAndValidateImage(txtPath, 0)
	if err == nil {
		t.Fatalf("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") && !strings.Contains(err.Error(), "format") {
		t.Errorf("expected format error message, got: %v", err)
	}
}

func TestLoadAndValidateImage_ExceedsSize(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "large.png")
	pngBytes := createTestPNG(16, 16)
	if err := os.WriteFile(pngPath, pngBytes, 0644); err != nil {
		t.Fatalf("failed to write png: %v", err)
	}

	// Set maxSize smaller than file size
	_, _, err := imgutil.LoadAndValidateImage(pngPath, 10)
	if err == nil {
		t.Fatalf("expected size error, got nil")
	}
	if !strings.Contains(err.Error(), "size") && !strings.Contains(err.Error(), "exceed") {
		t.Errorf("expected size limit error message, got: %v", err)
	}
}

func TestLoadAndValidateImage_RemoteURL(t *testing.T) {
	pngBytes := createTestPNG(32, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pngBytes)
	}))
	defer server.Close()

	data, mime, err := imgutil.LoadAndValidateImage(server.URL+"/image.png", 0)
	if err != nil {
		t.Fatalf("expected successful remote fetch, got error: %v", err)
	}
	if len(data) != len(pngBytes) {
		t.Errorf("expected %d bytes, got %d", len(pngBytes), len(data))
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got %s", mime)
	}
}

func TestGetDimensions(t *testing.T) {
	pngBytes := createTestPNG(256, 128)
	w, h, err := imgutil.GetDimensions(pngBytes)
	if err != nil {
		t.Fatalf("GetDimensions failed: %v", err)
	}
	if w != 256 || h != 128 {
		t.Errorf("expected 256x128, got %dx%d", w, h)
	}

	jpgBytes := createTestJPEG(320, 240)
	w, h, err = imgutil.GetDimensions(jpgBytes)
	if err != nil {
		t.Fatalf("GetDimensions JPEG failed: %v", err)
	}
	if w != 320 || h != 240 {
		t.Errorf("expected 320x240, got %dx%d", w, h)
	}

	// Invalid bytes
	_, _, err = imgutil.GetDimensions([]byte("corrupted"))
	if err == nil {
		t.Fatalf("expected error for corrupted bytes")
	}
}

func TestToBase64DataURI(t *testing.T) {
	pngBytes := createTestPNG(16, 16)
	uri := imgutil.ToBase64DataURI("image/png", pngBytes)
	if !strings.HasPrefix(uri, "data:image/png;base64,") {
		t.Errorf("expected data URI prefix, got: %s", uri[:30])
	}

	// Auto-detect MIME if empty
	uriAuto := imgutil.ToBase64DataURI("", pngBytes)
	if !strings.HasPrefix(uriAuto, "data:image/png;base64,") {
		t.Errorf("expected auto-detected image/png data URI, got: %s", uriAuto[:30])
	}
}

func TestValidateMatchingDimensions(t *testing.T) {
	base512 := createTestPNG(512, 512)
	mask512 := createTestPNG(512, 512)
	mask256 := createTestPNG(256, 256)

	// Matching dimensions
	if err := imgutil.ValidateMatchingDimensions(base512, mask512); err != nil {
		t.Errorf("expected matching dimensions to pass, got: %v", err)
	}

	// Mismatched dimensions
	err := imgutil.ValidateMatchingDimensions(base512, mask256)
	if err == nil {
		t.Fatalf("expected error for mismatched dimensions, got nil")
	}
	if !strings.Contains(err.Error(), "mismatch") && !strings.Contains(err.Error(), "512") {
		t.Errorf("expected descriptive dimension mismatch error, got: %v", err)
	}
}

func TestValidateMask(t *testing.T) {
	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "base.png")
	maskPath := filepath.Join(tmpDir, "mask.png")
	badMaskPath := filepath.Join(tmpDir, "bad_mask.png")

	_ = os.WriteFile(basePath, createTestPNG(200, 200), 0644)
	_ = os.WriteFile(maskPath, createTestPNG(200, 200), 0644)
	_ = os.WriteFile(badMaskPath, createTestPNG(100, 100), 0644)

	// Empty mask is no-op
	if err := imgutil.ValidateMask(basePath, "", 0); err != nil {
		t.Errorf("expected nil for empty mask, got %v", err)
	}

	// Mask without base
	if err := imgutil.ValidateMask("", maskPath, 0); err == nil {
		t.Errorf("expected error for mask without base")
	}

	// Valid matching mask
	if err := imgutil.ValidateMask(basePath, maskPath, 0); err != nil {
		t.Errorf("expected valid mask, got: %v", err)
	}

	// Mismatched mask
	if err := imgutil.ValidateMask(basePath, badMaskPath, 0); err == nil {
		t.Errorf("expected mismatch error, got nil")
	}
}

