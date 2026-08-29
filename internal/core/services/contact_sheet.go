package services

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ContactSheetExporter handles exporting batch results to JSON, Markdown, and HTML.
type ContactSheetExporter struct {
	OutputDir string
}

// NewContactSheetExporter creates a new ContactSheetExporter.
func NewContactSheetExporter(outputDir string) *ContactSheetExporter {
	return &ContactSheetExporter{
		OutputDir: outputDir,
	}
}

// Export writes batch_meta.json, summary.md, and index.html to the batch output directory.
func (e *ContactSheetExporter) Export(summary *BatchSummary) error {
	targetDir := e.OutputDir
	if targetDir == "" {
		targetDir = summary.OutputDir
	}
	if targetDir == "" {
		targetDir = filepath.Join("outputs", summary.BatchID)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", targetDir, err)
	}

	// 1. Export JSON
	jsonData, err := e.ExportJSON(summary)
	if err != nil {
		return fmt.Errorf("export JSON: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "batch_meta.json"), []byte(jsonData), 0o644); err != nil {
		return fmt.Errorf("write batch_meta.json: %w", err)
	}

	// 2. Export Markdown
	mdData, err := e.ExportMarkdown(summary)
	if err != nil {
		return fmt.Errorf("export Markdown: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "summary.md"), []byte(mdData), 0o644); err != nil {
		return fmt.Errorf("write summary.md: %w", err)
	}

	// 3. Export HTML
	htmlData, err := e.ExportHTML(summary)
	if err != nil {
		return fmt.Errorf("export HTML: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "index.html"), []byte(htmlData), 0o644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}

	return nil
}

// ExportJSON serializes the summary and telemetry into a formatted JSON string.
func (e *ContactSheetExporter) ExportJSON(summary *BatchSummary) (string, error) {
	type jobEntry struct {
		Index          int      `json:"index"`
		ID             string   `json:"id"`
		Prompt         string   `json:"prompt"`
		Seed           int64    `json:"seed"`
		Backend        string   `json:"backend"`
		Model          string   `json:"model"`
		AspectRatio    string   `json:"aspect_ratio"`
		Status         string   `json:"status"`
		DurationMs     int64    `json:"duration_ms"`
		ImageSizeBytes int64    `json:"image_size_bytes,omitempty"`
		ImagePath      string   `json:"image_path,omitempty"`
		Resolution     string   `json:"resolution,omitempty"`
		CriticScore    *float64 `json:"critic_score,omitempty"`
		CriticNotes    string   `json:"critic_notes,omitempty"`
		Error          string   `json:"error,omitempty"`
	}

	var jobs []jobEntry
	for _, r := range summary.Results {
		jobs = append(jobs, jobEntry{
			Index:          r.Job.Index,
			ID:             r.Job.ID,
			Prompt:         r.Job.Prompt,
			Seed:           r.Job.Seed,
			Backend:        r.Job.Backend,
			Model:          r.Job.Model,
			AspectRatio:    string(r.Job.AspectRatio),
			Status:         r.Status,
			DurationMs:     r.DurationMs,
			ImageSizeBytes: r.ImageSizeBytes,
			ImagePath:      r.ImagePath,
			Resolution:     r.Resolution,
			CriticScore:    r.CriticScore,
			CriticNotes:    r.CriticNotes,
			Error:          r.Error,
		})
	}

	payload := map[string]any{
		"batch_id":          summary.BatchID,
		"created_at":        summary.CreatedAt.Format(time.RFC3339),
		"total_jobs":        summary.TotalJobs,
		"success_count":     summary.SuccessCount,
		"failed_count":      summary.FailedCount,
		"total_duration_ms": summary.TotalDurationMs,
		"config":            summary.Config,
		"jobs":              jobs,
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ExportMarkdown builds a formatted Markdown table and aggregate statistics.
func (e *ContactSheetExporter) ExportMarkdown(summary *BatchSummary) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Batch Generation Summary: `%s`\n\n", summary.BatchID))
	sb.WriteString(fmt.Sprintf("- **Created**: %s\n", summary.CreatedAt.Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("- **Total Jobs**: %d (✅ %d Success, ❌ %d Failed)\n",
		summary.TotalJobs, summary.SuccessCount, summary.FailedCount))
	sb.WriteString(fmt.Sprintf("- **Total Duration**: %v\n\n", summary.TotalDuration.Round(time.Millisecond)))

	sb.WriteString("## Generation Results\n\n")
	sb.WriteString("| Job | Backend | Seed | Status | Duration | Size | Score | Prompt |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|\n")

	type backendStat struct {
		count    int
		success  int
		failed   int
		totalDur time.Duration
	}
	backendStats := make(map[string]*backendStat)

	for _, r := range summary.Results {
		stat, exists := backendStats[r.Job.Backend]
		if !exists {
			stat = &backendStat{}
			backendStats[r.Job.Backend] = stat
		}
		stat.count++
		stat.totalDur += r.Duration
		if r.Status == "SUCCESS" {
			stat.success++
		} else {
			stat.failed++
		}

		durStr := fmt.Sprintf("%.2fs", r.Duration.Seconds())
		sizeStr := "-"
		if r.ImageSizeBytes > 0 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(r.ImageSizeBytes)/1024.0)
		}
		scoreStr := "-"
		if r.CriticScore != nil {
			scoreStr = fmt.Sprintf("%.2f", *r.CriticScore)
		}

		escapedPrompt := strings.ReplaceAll(r.Job.Prompt, "|", "\\|")
		sb.WriteString(fmt.Sprintf("| %d | %s | %d | %s | %s | %s | %s | %s |\n",
			r.Job.Index, r.Job.Backend, r.Job.Seed, r.Status, durStr, sizeStr, scoreStr, escapedPrompt))
	}

	sb.WriteString("\n## Aggregate Performance\n\n")
	sb.WriteString("| Backend | Total Jobs | Success Rate | Avg Duration |\n")
	sb.WriteString("|---|---|---|---|\n")

	for bName, stat := range backendStats {
		successRate := float64(stat.success) / float64(stat.count) * 100.0
		avgDur := stat.totalDur / time.Duration(stat.count)
		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% (%d/%d) | %.2fs |\n",
			bName, stat.count, successRate, stat.success, stat.count, avgDur.Seconds()))
	}

	return sb.String(), nil
}

// ExportHTML produces a standalone HTML5 contact sheet with dark cyberpunk theme.
func (e *ContactSheetExporter) ExportHTML(summary *BatchSummary) (string, error) {
	var cards strings.Builder

	for _, r := range summary.Results {
		durStr := fmt.Sprintf("%.2fs", r.Duration.Seconds())
		scoreBadge := ""
		if r.CriticScore != nil {
			scoreBadge = fmt.Sprintf(`<span class="badge score">👁️ Critic: %.2f</span>`, *r.CriticScore)
		}

		if r.Status == "SUCCESS" {
			// Find relative image path or link
			imgRelPath := r.ImagePath
			if rel, err := filepath.Rel(summary.OutputDir, r.ImagePath); err == nil && !strings.HasPrefix(rel, "..") {
				imgRelPath = rel
			}

			cards.WriteString(fmt.Sprintf(`
		<div class="card success">
			<div class="card-img-wrapper">
				<a href="%s" target="_blank">
					<img src="%s" alt="%s" loading="lazy" />
				</a>
			</div>
			<div class="card-body">
				<div class="card-meta">
					<span class="badge backend">%s</span>
					<span class="badge seed">🌱 %d</span>
					<span class="badge dur">⏱️ %s</span>
					%s
				</div>
				<p class="prompt">%s</p>
			</div>
		</div>`,
				html.EscapeString(imgRelPath),
				html.EscapeString(imgRelPath),
				html.EscapeString(r.Job.Prompt),
				html.EscapeString(r.Job.Backend),
				r.Job.Seed,
				durStr,
				scoreBadge,
				html.EscapeString(r.Job.Prompt),
			))
		} else {
			cards.WriteString(fmt.Sprintf(`
		<div class="card failed">
			<div class="card-body">
				<div class="card-meta">
					<span class="badge backend">%s</span>
					<span class="badge seed">🌱 %d</span>
					<span class="badge status-fail">FAILED</span>
					<span class="badge dur">⏱️ %s</span>
				</div>
				<div class="error-box">
					<strong>Error:</strong> %s
				</div>
				<p class="prompt">%s</p>
			</div>
		</div>`,
				html.EscapeString(r.Job.Backend),
				r.Job.Seed,
				durStr,
				html.EscapeString(r.Error),
				html.EscapeString(r.Job.Prompt),
			))
		}
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>ARIS Batch Contact Sheet - %s</title>
	<style>
		:root {
			--bg-primary: #0b0f19;
			--bg-card: #151d30;
			--border: #223254;
			--text-main: #e2e8f0;
			--text-muted: #94a3b8;
			--accent: #38bdf8;
			--success: #10b981;
			--failed: #ef4444;
			--seed: #f59e0b;
		}
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body {
			background: var(--bg-primary);
			color: var(--text-main);
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			padding: 2rem;
		}
		header {
			margin-bottom: 2rem;
			border-bottom: 1px solid var(--border);
			padding-bottom: 1.5rem;
		}
		h1 { font-size: 1.75rem; color: var(--accent); margin-bottom: 0.5rem; }
		.summary-bar {
			display: flex;
			gap: 1.5rem;
			flex-wrap: wrap;
			font-size: 0.95rem;
			color: var(--text-muted);
		}
		.summary-bar strong { color: var(--text-main); }
		.grid {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
			gap: 1.5rem;
		}
		.card {
			background: var(--bg-card);
			border: 1px solid var(--border);
			border-radius: 8px;
			overflow: hidden;
			display: flex;
			flex-direction: column;
			transition: transform 0.2s, border-color 0.2s;
		}
		.card:hover {
			transform: translateY(-2px);
			border-color: var(--accent);
		}
		.card-img-wrapper {
			width: 100%%;
			aspect-ratio: 1 / 1;
			background: #000;
			overflow: hidden;
		}
		.card-img-wrapper img {
			width: 100%%;
			height: 100%%;
			object-fit: cover;
			display: block;
		}
		.card-body { padding: 1rem; display: flex; flex-direction: column; gap: 0.75rem; flex: 1; }
		.card-meta { display: flex; flex-wrap: wrap; gap: 0.4rem; font-size: 0.75rem; }
		.badge {
			padding: 0.2rem 0.5rem;
			border-radius: 4px;
			font-weight: 600;
			font-family: monospace;
			background: rgba(255, 255, 255, 0.05);
			border: 1px solid var(--border);
		}
		.badge.backend { color: var(--accent); border-color: rgba(56, 189, 248, 0.3); }
		.badge.seed { color: var(--seed); border-color: rgba(245, 158, 11, 0.3); }
		.badge.score { color: var(--success); border-color: rgba(16, 185, 129, 0.3); }
		.badge.status-fail { color: var(--failed); background: rgba(239, 68, 68, 0.1); border-color: var(--failed); }
		.prompt {
			font-size: 0.85rem;
			line-height: 1.4;
			color: var(--text-muted);
			word-break: break-word;
		}
		.error-box {
			background: rgba(239, 68, 68, 0.1);
			border-left: 3px solid var(--failed);
			padding: 0.5rem;
			font-size: 0.8rem;
			color: #fca5a5;
		}
	</style>
</head>
<body>
	<header>
		<h1>🎨 ARIS Batch Contact Sheet</h1>
		<div class="summary-bar">
			<div>Batch ID: <strong>%s</strong></div>
			<div>Total Jobs: <strong>%d</strong></div>
			<div>Success: <strong style="color: var(--success)">%d</strong></div>
			<div>Failed: <strong style="color: var(--failed)">%d</strong></div>
			<div>Duration: <strong>%v</strong></div>
		</div>
	</header>
	<main class="grid">
		%s
	</main>
</body>
</html>`

	fullHTML := fmt.Sprintf(htmlTemplate,
		summary.BatchID,
		summary.BatchID,
		summary.TotalJobs,
		summary.SuccessCount,
		summary.FailedCount,
		summary.TotalDuration.Round(time.Millisecond),
		cards.String(),
	)

	return fullHTML, nil
}
