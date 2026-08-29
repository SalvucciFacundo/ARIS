package imgutil

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MaxImageSize defines the default maximum allowed image size (25 MB).
const MaxImageSize int64 = 25 * 1024 * 1024

// DetectMIME inspects magic bytes to detect standard image formats (PNG, JPEG, WEBP).
func DetectMIME(data []byte) (string, error) {
	if len(data) < 12 {
		return "", fmt.Errorf("file too short to determine image format")
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return "image/png", nil
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg", nil
	}

	// WEBP: RIFF....WEBP
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && string(data[8:12]) == "WEBP" {
		return "image/webp", nil
	}

	return "", fmt.Errorf("unsupported image format (expected PNG, JPEG, or WEBP)")
}

// LoadAndValidateImage reads and validates an image from a local path or HTTP/HTTPS URL.
// It enforces format validation (PNG, JPEG, WEBP) and maximum payload bounds (default 25 MB).
func LoadAndValidateImage(pathOrURL string, maxSize int64) ([]byte, string, error) {
	if maxSize <= 0 {
		maxSize = MaxImageSize
	}

	trimmed := strings.TrimSpace(pathOrURL)
	if trimmed == "" {
		return nil, "", fmt.Errorf("image path or URL cannot be empty")
	}

	var data []byte
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest(http.MethodGet, trimmed, nil)
		if err != nil {
			return nil, "", fmt.Errorf("create HTTP request: %w", err)
		}
		req.Header.Set("User-Agent", "ARIS-Agent/1.0")

		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("fetch remote image: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("remote server returned status %d", resp.StatusCode)
		}

		lr := io.LimitReader(resp.Body, maxSize+1)
		b, err := io.ReadAll(lr)
		if err != nil {
			return nil, "", fmt.Errorf("read remote image payload: %w", err)
		}
		if int64(len(b)) > maxSize {
			return nil, "", fmt.Errorf("image payload exceeds maximum allowed size of %d bytes", maxSize)
		}
		data = b
	} else {
		info, err := os.Stat(trimmed)
		if err != nil {
			return nil, "", fmt.Errorf("open image file %q: %w", trimmed, err)
		}
		if info.IsDir() {
			return nil, "", fmt.Errorf("path %q is a directory, expected image file", trimmed)
		}
		if info.Size() > maxSize {
			return nil, "", fmt.Errorf("image file size (%d bytes) exceeds maximum allowed size of %d bytes", info.Size(), maxSize)
		}

		b, err := os.ReadFile(trimmed)
		if err != nil {
			return nil, "", fmt.Errorf("read image file: %w", err)
		}
		data = b
	}

	mimeType, err := DetectMIME(data)
	if err != nil {
		return nil, "", err
	}

	return data, mimeType, nil
}

// GetDimensions extracts width and height from raw image bytes.
func GetDimensions(data []byte) (width, height int, err error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("empty image data")
	}

	// Try standard image.DecodeConfig first (PNG, JPEG, GIF)
	cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
	if decodeErr == nil && cfg.Width > 0 && cfg.Height > 0 {
		return cfg.Width, cfg.Height, nil
	}

	// Handle WEBP headers manually
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 30 && string(data[8:12]) == "WEBP" {
		chunkType := string(data[12:16])
		switch chunkType {
		case "VP8 ": // Lossy
			if len(data) >= 30 {
				w := int(binary.LittleEndian.Uint16(data[26:28]) & 0x3fff)
				h := int(binary.LittleEndian.Uint16(data[28:30]) & 0x3fff)
				return w, h, nil
			}
		case "VP8L": // Lossless
			if len(data) >= 25 && data[20] == 0x2f {
				b1, b2, b3, b4 := uint32(data[21]), uint32(data[22]), uint32(data[23]), uint32(data[24])
				val := b1 | (b2 << 8) | (b3 << 16) | (b4 << 24)
				w := int(val&0x3fff) + 1
				h := int((val>>14)&0x3fff) + 1
				return w, h, nil
			}
		case "VP8X": // Extended
			if len(data) >= 30 {
				w := int(data[24]) | int(data[25])<<8 | int(data[26])<<16 + 1
				h := int(data[27]) | int(data[28])<<8 | int(data[29])<<16 + 1
				return w, h, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("unable to decode image dimensions: %w", decodeErr)
}

// ToBase64DataURI encodes raw image bytes as a data URI (e.g. data:image/png;base64,...).
func ToBase64DataURI(mimeType string, data []byte) string {
	if mimeType == "" {
		detected, err := DetectMIME(data)
		if err == nil {
			mimeType = detected
		} else {
			mimeType = "image/png"
		}
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// ValidateMatchingDimensions verifies that the base image and mask have identical width and height.
func ValidateMatchingDimensions(imgData, maskData []byte) error {
	w1, h1, err := GetDimensions(imgData)
	if err != nil {
		return fmt.Errorf("invalid base image: %w", err)
	}

	w2, h2, err := GetDimensions(maskData)
	if err != nil {
		return fmt.Errorf("invalid mask image: %w", err)
	}

	if w1 != w2 || h1 != h2 {
		return fmt.Errorf("dimension mismatch: base image is %dx%d, but mask image is %dx%d", w1, h1, w2, h2)
	}

	return nil
}
