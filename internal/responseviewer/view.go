package responseviewer

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/ui"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string

	switch {
	case m.loading:
		content = ui.EmptyState("Sending request...", m.width-2, m.height-3)
	case !m.hasResp:
		content = ui.EmptyState("Send a request to see the response", m.width-2, m.height-3)
	default:
		content = m.viewport.View()
	}

	return tea.NewView(m.applyBorder(content))
}

func (m Model) applyBorder(content string) string {
	border := ui.PaneBorder
	if m.focused {
		border = ui.PaneFocusedBorder
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	title := titleStyle.Render(" Response ")

	return border.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(title + "\n" + content)
}
