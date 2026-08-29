package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full split screen Cyberpunk interface.
func (m Model) View() string {
	if !m.ready {
		return "⚡ Initializing ARIS Cyberpunk TUI..."
	}

	leftWidth := m.width * 58 / 100
	rightWidth := m.width - leftWidth - 5
	if rightWidth < 30 {
		rightWidth = 30
	}

	// 1. Header
	header := HeaderStyle.Width(m.width - 2).Render(
		fmt.Sprintf("⚡ ARIS: Autonomous Reasoner for Image System  │  🔌 Backend: %s  │  📐 Ratio: %s  │  🌱 Model: %s",
			m.activeBackend, m.activeRatio, m.activeModel),
	)

	// 2. Left Pane (Chat history + textarea + spinner)
	var leftContent strings.Builder
	leftContent.WriteString(m.viewport.View())
	leftContent.WriteString("\n")

	if m.isGenerating {
		leftContent.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), m.statusText))
	} else {
		leftContent.WriteString(m.textarea.View() + "\n")
	}

	leftPane := LeftPaneStyle.Width(leftWidth).Height(m.height - 6).Render(leftContent.String())

	// 3. Right Pane (Parameter knobs + Memory Facts + ANSI Image Preview)
	var rightContent strings.Builder
	rightContent.WriteString(BadgeStyle.Render("INSPECTOR & CONTROLS") + "\n\n")

	rightContent.WriteString(KnobLabelStyle.Render("Backend: ") + KnobValueStyle.Render(m.activeBackend) + " (Ctrl+B to cycle)\n")
	rightContent.WriteString(KnobLabelStyle.Render("Ratio:   ") + KnobValueStyle.Render(string(m.activeRatio)) + " (Tab to cycle)\n")
	rightContent.WriteString(KnobLabelStyle.Render("Model:   ") + KnobValueStyle.Render(m.activeModel) + "\n\n")

	// Recalled / Learned Knowledge Graph Facts
	rightContent.WriteString(KnobLabelStyle.Render("🧠 Knowledge Graph Memory:") + "\n")
	if len(m.facts) == 0 {
		rightContent.WriteString(FactStyle.Render("  (Learning loop active — no facts yet)") + "\n")
	} else {
		for i, f := range m.facts {
			if i >= 3 {
				break
			}
			factText := f.Fact
			if len(factText) > 40 {
				factText = factText[:37] + "..."
			}
			rightContent.WriteString(fmt.Sprintf("  • [%s] %s\n", f.Scope, factText))
		}
	}
	rightContent.WriteString("\n")

	// Terminal Image Preview
	rightContent.WriteString(KnobLabelStyle.Render("🖼️ Rendered Image Preview:") + "\n")
	if m.ansiPreview != "" {
		rightContent.WriteString(m.ansiPreview + "\n")
		rightContent.WriteString(FactStyle.Render("Press Ctrl+O to open full-res image") + "\n")
	} else {
		rightContent.WriteString(FactStyle.Render("  [No image rendered in this session]") + "\n")
	}

	rightPane := RightPaneStyle.Width(rightWidth).Height(m.height - 6).Render(rightContent.String())

	// 4. Main Body
	mainLayout := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// 5. Footer Status Bar
	footer := HeaderStyle.Width(m.width - 2).Render(
		fmt.Sprintf("STATUS: %s  │  Enter: Send  │  Tab: Ratio  │  Ctrl+B: Backend  │  Ctrl+O: Open Image  │  Esc: Exit", m.statusText),
	)

	return lipgloss.JoinVertical(lipgloss.Left, header, mainLayout, footer)
}
