package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createTestPNG(t *testing.T, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	return buf.Bytes()
}

func TestHandlers_Endpoints(t *testing.T) {
	broker := NewSSEBroker()
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     8080,
		AutoPort: true,
	}
	h := NewHandlers(nil, broker, cfg)
	router := NewRouter(cfg, h, broker)

	t.Run("GET / index HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "ARIS") {
			t.Errorf("expected body to contain ARIS, got %s", body)
		}
	})

	t.Run("GET /api/subagents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subagents", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var subs []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &subs); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if len(subs) == 0 {
			t.Error("expected non-empty subagents list")
		}
	})

	t.Run("GET /api/backends", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/backends", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var backends []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &backends); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if len(backends) == 0 {
			t.Error("expected non-empty backends list")
		}
	})

	t.Run("GET /api/history", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/history?limit=10&offset=0", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var items []any
		if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
	})

	t.Run("POST /api/generate validation", func(t *testing.T) {
		// Empty prompt should fail
		payload := `{"prompt": ""}`
		req := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(payload))
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for empty prompt, got %d", rec.Code)
		}

		// Valid prompt
		validPayload := `{"prompt": "a cyber cat", "aspect_ratio": "16:9", "backend": "pollinations"}`
		req2 := httptest.NewRequest(http.MethodPost, "/api/generate", strings.NewReader(validPayload))
		req2.RemoteAddr = "127.0.0.1:1234"
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d", rec2.Code)
		}
	})

	t.Run("POST /api/inpaint multipart", func(t *testing.T) {
		imgBytes := createTestPNG(t, 64, 64)
		maskBytes := createTestPNG(t, 64, 64)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		_ = writer.WriteField("prompt", "inpaint neon lights")
		_ = writer.WriteField("backend", "pollinations")

		imgPart, _ := writer.CreateFormFile("image", "base.png")
		_, _ = imgPart.Write(imgBytes)

		maskPart, _ := writer.CreateFormFile("mask", "mask.png")
		_, _ = maskPart.Write(maskBytes)

		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/inpaint", &buf)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}
