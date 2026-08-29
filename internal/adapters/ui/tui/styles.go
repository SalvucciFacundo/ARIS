package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Cyberpunk Palette
	ColorCyan    = lipgloss.Color("#00f0ff")
	ColorMagenta = lipgloss.Color("#ff007f")
	ColorGreen   = lipgloss.Color("#00ff9f")
	ColorYellow  = lipgloss.Color("#ffe600")
	ColorPurple  = lipgloss.Color("#9d00ff")
	ColorDarkBg  = lipgloss.Color("#08080c")
	ColorPanelBg = lipgloss.Color("#12121e")
	ColorBorder  = lipgloss.Color("#2a2a40")
	ColorDim     = lipgloss.Color("#626280")
	ColorWhite   = lipgloss.Color("#e0e0ff")

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			Background(ColorPanelBg).
			Padding(0, 1).
			MarginBottom(1)

	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDarkBg).
			Background(ColorMagenta).
			Padding(0, 1)

	// Panels
	LeftPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorCyan).
			Padding(0, 1)

	RightPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMagenta).
			Padding(0, 1)

	// Messages
	UserMsgStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorYellow).
			MarginTop(1)

	AgentThoughtStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(ColorDim).
				MarginBottom(1)

	AgentMsgStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			MarginBottom(1)

	SpecCardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorGreen).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	// Status & Knobs
	KnobLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPurple)

	KnobValueStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	FactStyle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Italic(true)
)
