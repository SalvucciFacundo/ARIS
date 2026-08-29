package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
	"aris/pkg/imgutil"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatMessage struct {
	role    string // "user", "agent", "thought", "system"
	content string
	spec    *domain.ImageSpec
	result  *domain.ImageResult
}

type genCompletedMsg struct {
	spec   *domain.ImageSpec
	result *domain.ImageResult
	err    error
}

type previewLoadedMsg struct {
	ansi string
}

type Model struct {
	agent         *services.AgentService
	viewport      viewport.Model
	textarea      textarea.Model
	spinner       spinner.Model
	messages      []chatMessage
	isGenerating  bool
	statusText    string
	activeBackend string
	activeRatio   domain.AspectRatio
	activeModel   string
	lastSpec      *domain.ImageSpec
	lastResult    *domain.ImageResult
	ansiPreview   string
	facts         []domain.KnowledgeFact
	width         int
	height        int
	ready         bool
}

// NewModel creates an initialized TUI state model.
func NewModel(agent *services.AgentService) Model {
	ta := textarea.New()
	ta.Placeholder = "Describe an image or give follow-up instructions... (Enter to generate)"
	ta.Focus()
	ta.Prompt = "⚡ "
	ta.CharLimit = 500
	ta.SetWidth(60)
	ta.SetHeight(2)
	ta.FocusedStyle.CursorLine = ta.FocusedStyle.CursorLine.Foreground(ColorCyan)

	sp := spinner.New()
	sp.Spinner = spinner.Pulse
	sp.Style = lipglossStyleSpinner()

	defBackend := "pollinations"
	if agent.Registry() != nil && agent.Registry().GetDefault() != nil {
		defBackend = agent.Registry().GetDefault().Name()
	}

	return Model{
		agent:         agent,
		textarea:      ta,
		spinner:       sp,
		messages:      make([]chatMessage, 0),
		activeBackend: defBackend,
		activeRatio:   domain.RatioSquare,
		activeModel:   "flux",
		statusText:    "Ready. Type prompt and press Enter.",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.loadRecentFactsCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyCtrlO:
			if m.lastResult != nil && m.lastResult.LocalPath != "" {
				_ = imgutil.OpenInViewer(m.lastResult.LocalPath)
				m.statusText = fmt.Sprintf("Opened in system viewer: %s", m.lastResult.LocalPath)
			}

		case tea.KeyCtrlL:
			m.messages = nil
			m.viewport.SetContent("")
			m.statusText = "Chat cleared."

		case tea.KeyTab:
			m.cycleAspectRatio()

		case tea.KeyCtrlB:
			m.cycleBackend()

		case tea.KeyEnter:
			if m.isGenerating {
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			m.textarea.Reset()
			m.messages = append(m.messages, chatMessage{
				role:    "user",
				content: input,
			})
			m.isGenerating = true
			m.statusText = fmt.Sprintf("Reasoning prompt with Art Director & dispatching to %s...", m.activeBackend)
			m.updateViewportContent()

			return m, tea.Batch(
				m.spinner.Tick,
				m.generateImageCmd(input),
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width*6/10, msg.Height-8)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width * 6 / 10
			m.viewport.Height = msg.Height - 8
		}
		m.textarea.SetWidth(m.viewport.Width - 4)
		m.updateViewportContent()

	case genCompletedMsg:
		m.isGenerating = false
		if msg.err != nil {
			m.statusText = fmt.Sprintf("Error: %v", msg.err)
			m.messages = append(m.messages, chatMessage{
				role:    "system",
				content: fmt.Sprintf("❌ Generation failed: %v", msg.err),
			})
		} else {
			m.lastSpec = msg.spec
			m.lastResult = msg.result
			m.statusText = fmt.Sprintf("✨ Generated in %v. Image saved to %s",
				msg.result.Duration.Round(time.Millisecond), msg.result.LocalPath)

			m.messages = append(m.messages, chatMessage{
				role:    "agent",
				content: msg.spec.EnhancedPrompt,
				spec:    msg.spec,
				result:  msg.result,
			})

			// Render ANSI preview
			cmds = append(cmds, m.loadPreviewCmd(msg.result.LocalPath))
			cmds = append(cmds, m.loadRecentFactsCmd())
		}
		m.updateViewportContent()

	case previewLoadedMsg:
		m.ansiPreview = msg.ansi

	case []domain.KnowledgeFact:
		m.facts = msg
	}

	if m.isGenerating {
		m.spinner, spCmd = m.spinner.Update(msg)
		cmds = append(cmds, spCmd)
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) cycleAspectRatio() {
	switch m.activeRatio {
	case domain.RatioSquare:
		m.activeRatio = domain.RatioLandscape
	case domain.RatioLandscape:
		m.activeRatio = domain.RatioPortrait
	case domain.RatioPortrait:
		m.activeRatio = domain.RatioWide
	case domain.RatioWide:
		m.activeRatio = domain.RatioPhoto
	default:
		m.activeRatio = domain.RatioSquare
	}
	m.statusText = fmt.Sprintf("Aspect ratio changed to %s", m.activeRatio)
}

func (m *Model) cycleBackend() {
	if m.agent.Registry() == nil {
		return
	}
	list := m.agent.Registry().List()
	if len(list) == 0 {
		return
	}
	currIdx := 0
	for i, name := range list {
		if name == m.activeBackend {
			currIdx = i
			break
		}
	}
	nextIdx := (currIdx + 1) % len(list)
	m.activeBackend = list[nextIdx]
	m.statusText = fmt.Sprintf("Backend switched to %s", m.activeBackend)
}

func (m *Model) updateViewportContent() {
	var b strings.Builder
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			b.WriteString(UserMsgStyle.Render("👤 You: ") + msg.content + "\n\n")
		case "agent":
			b.WriteString(AgentMsgStyle.Render("🎨 ARIS (Art Director):") + "\n")
			if msg.spec != nil {
				card := fmt.Sprintf("📐 %dx%d (Ratio: %s) | 🌱 Seed: %d | 🔌 Backend: %s\n✨ Prompt: %s",
					msg.spec.Width, msg.spec.Height, msg.spec.AspectRatio, msg.spec.Seed, msg.spec.Backend, msg.spec.EnhancedPrompt)
				if msg.spec.NegativePrompt != "" {
					card += "\n🚫 Negative: " + msg.spec.NegativePrompt
				}
				b.WriteString(SpecCardStyle.Render(card) + "\n")
			}
			if msg.result != nil {
				b.WriteString(FactStyle.Render(fmt.Sprintf("💾 Saved: %s (%v)\n", msg.result.LocalPath, msg.result.Duration.Round(time.Millisecond))))
			}
			b.WriteString("\n")
		case "system":
			b.WriteString(msg.content + "\n\n")
		}
	}
	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

func (m Model) generateImageCmd(input string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		opts := services.GenerateOptions{
			AspectRatio: m.activeRatio,
			Backend:     m.activeBackend,
			Model:       m.activeModel,
		}

		spec, result, err := m.agent.Generate(ctx, input, opts)
		return genCompletedMsg{
			spec:   spec,
			result: result,
			err:    err,
		}
	}
}

func (m Model) loadPreviewCmd(imagePath string) tea.Cmd {
	return func() tea.Msg {
		ansi, err := imgutil.RenderANSIHalfBlocks(imagePath, 36, 16)
		if err != nil {
			return previewLoadedMsg{ansi: fmt.Sprintf("(Preview unavailable: %v)", err)}
		}
		return previewLoadedMsg{ansi: ansi}
	}
}

func (m Model) loadRecentFactsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		facts, _ := m.agent.SearchMemory(ctx, "", "", 5)
		return facts
	}
}

func lipglossStyleSpinner() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorMagenta).Bold(true)
}
