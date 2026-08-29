package tui

import (
	"fmt"

	"aris/internal/core/services"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the interactive Cyberpunk TUI.
func Run(agent *services.AgentService) error {
	m := NewModel(agent)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui program error: %w", err)
	}
	return nil
}
