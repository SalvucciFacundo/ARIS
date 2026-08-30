package imgutil_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"aris/pkg/imgutil"
)

func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 1, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestInjectAndExtractMetadata_RoundTrip(t *testing.T) {
	pngBytes := makeTestPNG(t)

	meta := map[string]string{
		"prompt":     `{"1": {"class_type": "KSampler", "inputs": {"seed": 42}}}`,
		"workflow":   `{"nodes": [{"id": 1, "type": "KSampler"}], "links": []}`,
		"parameters": "cyberpunk alley, neon lights\nSteps: 20, Sampler: Euler, CFG: 7.0",
	}

	var injectedBuf bytes.Buffer
	err := imgutil.InjectPNGMetadata(bytes.NewReader(pngBytes), &injectedBuf, meta)
	if err != nil {
		t.Fatalf("InjectPNGMetadata failed: %v", err)
	}

	// Verify the injected image is still a valid PNG that decodes properly
	_, err = png.Decode(bytes.NewReader(injectedBuf.Bytes()))
	if err != nil {
		t.Fatalf("failed to decode injected PNG with standard decoder: %v", err)
	}

	// Extract metadata
	extracted, err := imgutil.ExtractPNGMetadata(bytes.NewReader(injectedBuf.Bytes()))
	if err != nil {
		t.Fatalf("ExtractPNGMetadata failed: %v", err)
	}

	for k, expectedVal := range meta {
		actualVal, ok := extracted[k]
		if !ok {
			t.Errorf("missing key %q in extracted metadata", k)
			continue
		}
		if actualVal != expectedVal {
			t.Errorf("key %q value mismatch:\nexpected: %s\ngot:      %s", k, expectedVal, actualVal)
		}
	}
}

func TestExtractMetadata_InvalidSignature(t *testing.T) {
	invalidBytes := []byte("not a png file - arbitrary data")

	_, err := imgutil.ExtractPNGMetadata(bytes.NewReader(invalidBytes))
	if !errors.Is(err, imgutil.ErrInvalidPNGSignature) {
		t.Fatalf("expected ErrInvalidPNGSignature, got: %v", err)
	}

	var out bytes.Buffer
	err = imgutil.InjectPNGMetadata(bytes.NewReader(invalidBytes), &out, map[string]string{"k": "v"})
	if !errors.Is(err, imgutil.ErrInvalidPNGSignature) {
		t.Fatalf("expected ErrInvalidPNGSignature on inject, got: %v", err)
	}
}

func TestExtractMetadata_NoMetadata(t *testing.T) {
	pngBytes := makeTestPNG(t)

	extracted, err := imgutil.ExtractPNGMetadata(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("ExtractPNGMetadata on clean PNG failed: %v", err)
	}
	if len(extracted) != 0 {
		t.Fatalf("expected empty metadata map, got %d items", len(extracted))
	}
}

func TestExtractMetadata_CorruptedCRC(t *testing.T) {
	pngBytes := makeTestPNG(t)
	meta := map[string]string{"test": "val"}

	var injected bytes.Buffer
	if err := imgutil.InjectPNGMetadata(bytes.NewReader(pngBytes), &injected, meta); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	data := injected.Bytes()
	// Find the 'tEXt' chunk type and corrupt its CRC (the 4 bytes right after chunk data)
	idx := bytes.Index(data, []byte("tEXt"))
	if idx == -1 {
		t.Fatalf("could not find tEXt chunk in injected png")
	}

	// Chunk format: 4 bytes len, 4 bytes type, len bytes data, 4 bytes CRC
	chunkLen := binary.BigEndian.Uint32(data[idx-4 : idx])
	crcOffset := idx + 4 + int(chunkLen)
	// Corrupt CRC
	data[crcOffset] ^= 0xFF

	_, err := imgutil.ExtractPNGMetadata(bytes.NewReader(data))
	if !errors.Is(err, imgutil.ErrInvalidChunkCRC) {
		t.Fatalf("expected ErrInvalidChunkCRC, got: %v", err)
	}
}

func TestFileHelpers_RoundTrip(t *testing.T) {
	pngBytes := makeTestPNG(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.png")
	dstFile := filepath.Join(tmpDir, "injected.png")

	if err := os.WriteFile(srcFile, pngBytes, 0644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	meta := map[string]string{
		"workflow": `{"nodes": []}`,
		"prompt":   `{"3": {"class_type": "KSampler"}}`,
	}

	if err := imgutil.InjectPNGMetadataFile(srcFile, dstFile, meta); err != nil {
		t.Fatalf("InjectPNGMetadataFile failed: %v", err)
	}

	extracted, err := imgutil.ExtractPNGMetadataFile(dstFile)
	if err != nil {
		t.Fatalf("ExtractPNGMetadataFile failed: %v", err)
	}

	if extracted["workflow"] != meta["workflow"] || extracted["prompt"] != meta["prompt"] {
		t.Fatalf("file metadata round-trip mismatch: got %+v", extracted)
	}
}

func TestAliases_InjectAndExtractMetadata(t *testing.T) {
	pngBytes := makeTestPNG(t)
	meta := map[string]string{"alias_key": "alias_val"}

	var buf bytes.Buffer
	if err := imgutil.InjectMetadata(bytes.NewReader(pngBytes), &buf, meta); err != nil {
		t.Fatalf("InjectMetadata failed: %v", err)
	}

	extracted, err := imgutil.ExtractMetadata(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}

	if extracted["alias_key"] != "alias_val" {
		t.Fatalf("alias mismatch: %v", extracted)
	}
}

func TestInjectAndExtract_LargePayloadAndMultipleKeys(t *testing.T) {
	pngBytes := makeTestPNG(t)
	// Create a large 100KB json-like text
	largeVal := bytes.Repeat([]byte("abcdef1234567890"), 6400) // ~100KB

	meta := map[string]string{
		"key1": "short value",
		"key2": string(largeVal),
		"key3": "", // empty value test
	}

	var buf bytes.Buffer
	if err := imgutil.InjectPNGMetadata(bytes.NewReader(pngBytes), &buf, meta); err != nil {
		t.Fatalf("InjectPNGMetadata failed with large payload: %v", err)
	}

	extracted, err := imgutil.ExtractPNGMetadata(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ExtractPNGMetadata failed with large payload: %v", err)
	}

	if extracted["key1"] != "short value" {
		t.Errorf("key1 mismatch")
	}
	if extracted["key2"] != string(largeVal) {
		t.Errorf("key2 length mismatch: got %d, expected %d", len(extracted["key2"]), len(largeVal))
	}
	if extracted["key3"] != "" {
		t.Errorf("key3 empty value mismatch: got %q", extracted["key3"])
	}
}

