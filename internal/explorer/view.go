package explorer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/ui"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	if len(m.visible) == 0 {
		if len(m.warnings) > 0 {
			content := m.renderWarnings()
			return tea.NewView(m.applyBorder(content))
		}
		content := ui.EmptyState("No services loaded", m.width-4, m.height-4)
		return tea.NewView(m.applyBorder(content))
	}

	// Determine scroll offset to keep cursor visible.
	contentHeight := m.height - 2 // account for border
	if contentHeight < 1 {
		contentHeight = 1
	}
	scrollOffset := 0
	if m.cursor >= contentHeight {
		scrollOffset = m.cursor - contentHeight + 1
	}

	var lines []string
	for i, node := range m.visible {
		if i < scrollOffset {
			continue
		}
		if len(lines) >= contentHeight {
			break
		}

		line := m.renderNode(node, i == m.cursor)
		lines = append(lines, line)
	}

	// Pad remaining height.
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return tea.NewView(m.applyBorder(content))
}

func (m Model) renderNode(node *TreeNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)
	var line string

	switch node.Kind {
	case NodePackage:
		arrow := "▸"
		if node.Expanded {
			arrow = "▾"
		}
		line = fmt.Sprintf("%s%s %s", indent, arrow, node.Label)

	case NodeService:
		arrow := "▸"
		if node.Expanded {
			arrow = "▾"
		}
		line = fmt.Sprintf("%s%s %s", indent, arrow, node.Label)

	case NodeMethod:
		label := node.Label
		if node.IsStreaming {
			label += " [stream]"
		}
		line = fmt.Sprintf("%s  %s", indent, label)
	}

	// Truncate to fit width.
	maxWidth := m.width - 4 // account for border + padding
	if maxWidth > 0 && lipgloss.Width(line) > maxWidth {
		line = line[:maxWidth]
	}

	// Apply styling.
	if selected && m.focused {
		return ui.SelectedStyle.Render(padRight(line, maxWidth))
	}
	if selected {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.ColorText).
			Render(padRight(line, maxWidth))
	}
	if node.IsStreaming {
		return ui.DimmedStyle.Render(line)
	}

	return lipgloss.NewStyle().Foreground(ui.ColorText).Render(line)
}

func (m Model) renderWarnings() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWarning)
	errStyle := lipgloss.NewStyle().Foreground(ui.ColorError)

	var lines []string
	lines = append(lines, titleStyle.Render("Failed to load services"))
	lines = append(lines, "")
	for _, w := range m.warnings {
		maxLen := m.width - 8
		if maxLen < 20 {
			maxLen = 20
		}
		display := w
		if len(display) > maxLen {
			display = display[:maxLen-3] + "..."
		}
		lines = append(lines, errStyle.Render("  "+display))
	}
	lines = append(lines, "")
	lines = append(lines, ui.DimmedStyle.Render("Ctrl+R to reload | Ctrl+W for details"))

	content := strings.Join(lines, "\n")
	innerW := m.width - 4
	innerH := m.height - 4
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	return lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m Model) applyBorder(content string) string {
	border := ui.PaneBorder
	if m.focused {
		border = ui.PaneFocusedBorder
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	title := titleStyle.Render(" Explorer ")

	return border.
		Width(m.width - 2).
		Height(m.height - 2).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Render(title + "\n" + content)
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
