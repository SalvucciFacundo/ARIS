package services_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
)

func sampleSummary() *services.BatchSummary {
	score := 8.5
	return &services.BatchSummary{
		BatchID:         "batch_20260330_120000_abcd1234",
		CreatedAt:       time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
		TotalJobs:       2,
		SuccessCount:    1,
		FailedCount:     1,
		TotalDuration:   3500 * time.Millisecond,
		TotalDurationMs: 3500,
		OutputDir:       "/tmp/test_batch_out",
		Config: services.BatchConfig{
			Count:       2,
			Backends:    []string{"pollinations", "falai"},
			Model:       "flux",
			AspectRatio: domain.RatioSquare,
			Concurrency: 2,
		},
		Results: []services.BatchJobResult{
			{
				Job: services.BatchJob{
					ID:          "job-001",
					Index:       1,
					Prompt:      "a futuristic cityscape",
					Seed:        1001,
					Backend:     "pollinations",
					Model:       "flux",
					AspectRatio: domain.RatioSquare,
				},
				Status:         "SUCCESS",
				Duration:       1840 * time.Millisecond,
				DurationMs:     1840,
				ImageSizeBytes: 524288,
				ImagePath:      "/tmp/test_batch_out/images/job_1_pollinations.png",
				Resolution:     "1024x1024",
				CriticScore:    &score,
				CriticNotes:    "Sharp composition and vibrant lighting.",
			},
			{
				Job: services.BatchJob{
					ID:          "job-002",
					Index:       2,
					Prompt:      "a cyberpunk street",
					Seed:        1002,
					Backend:     "falai",
					Model:       "flux",
					AspectRatio: domain.RatioSquare,
				},
				Status:     "FAILED",
				Error:      "upstream gateway timeout (HTTP 504)",
				Duration:   1660 * time.Millisecond,
				DurationMs: 1660,
			},
		},
	}
}

func TestContactSheetExporter_ExportJSON(t *testing.T) {
	exporter := services.NewContactSheetExporter("")
	summary := sampleSummary()

	data, err := exporter.ExportJSON(summary)
	if err != nil {
		t.Fatalf("ExportJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		t.Fatalf("invalid JSON generated: %v", err)
	}

	if parsed["batch_id"] != "batch_20260330_120000_abcd1234" {
		t.Errorf("expected batch_id %q, got %v", "batch_20260330_120000_abcd1234", parsed["batch_id"])
	}
	if parsed["total_jobs"].(float64) != 2 {
		t.Errorf("expected 2 total_jobs, got %v", parsed["total_jobs"])
	}
	if parsed["success_count"].(float64) != 1 {
		t.Errorf("expected 1 success_count, got %v", parsed["success_count"])
	}
	if parsed["failed_count"].(float64) != 1 {
		t.Errorf("expected 1 failed_count, got %v", parsed["failed_count"])
	}
}

func TestContactSheetExporter_ExportMarkdown(t *testing.T) {
	exporter := services.NewContactSheetExporter("")
	summary := sampleSummary()

	md, err := exporter.ExportMarkdown(summary)
	if err != nil {
		t.Fatalf("ExportMarkdown error: %v", err)
	}

	requiredSnippets := []string{
		"# Batch Generation Summary",
		"| Job | Backend | Seed | Status | Duration | Size | Score | Prompt |",
		"| 1 | pollinations | 1001 | SUCCESS |",
		"| 2 | falai | 1002 | FAILED |",
		"## Aggregate Performance",
		"pollinations",
		"falai",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(md, snippet) {
			t.Errorf("expected markdown to contain snippet %q, but was missing:\n%s", snippet, md)
		}
	}
}

func TestContactSheetExporter_ExportHTML(t *testing.T) {
	exporter := services.NewContactSheetExporter("")
	summary := sampleSummary()

	html, err := exporter.ExportHTML(summary)
	if err != nil {
		t.Fatalf("ExportHTML error: %v", err)
	}

	requiredHTML := []string{
		"<!DOCTYPE html>",
		"batch_20260330_120000_abcd1234",
		"a futuristic cityscape",
		"a cyberpunk street",
		"upstream gateway timeout (HTTP 504)",
		"pollinations",
		"falai",
		"1001",
		"1002",
		"8.50",
	}

	for _, snippet := range requiredHTML {
		if !strings.Contains(html, snippet) {
			t.Errorf("expected HTML to contain snippet %q, but was missing:\n%s", snippet, html)
		}
	}
}

func TestContactSheetExporter_ExportDisk(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aris_batch_export_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exporter := services.NewContactSheetExporter(tmpDir)
	summary := sampleSummary()
	summary.OutputDir = tmpDir

	if err := exporter.Export(summary); err != nil {
		t.Fatalf("Export error: %v", err)
	}

	// Verify all 3 files exist
	jsonPath := filepath.Join(tmpDir, "batch_meta.json")
	mdPath := filepath.Join(tmpDir, "summary.md")
	htmlPath := filepath.Join(tmpDir, "index.html")

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("file %s was not created", jsonPath)
	}
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Errorf("file %s was not created", mdPath)
	}
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Errorf("file %s was not created", htmlPath)
	}
}
