package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// HelpOverlay renders a full-screen overlay showing all keybindings.
func HelpOverlay(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorText)
	descStyle := lipgloss.NewStyle().Foreground(ColorDimmed)

	var lines []string

	lines = append(lines, titleStyle.Render("Protopilot Keybindings"))
	lines = append(lines, "")

	lines = append(lines, sectionStyle.Render("Global"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Tab", "Cycle focus to next pane"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Shift+Tab", "Cycle focus to previous pane"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+Enter", "Send gRPC request"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+C", "Quit application"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "q", "Quit (when not in text input)"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "?", "Toggle this help overlay"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+R", "Reload proto files"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+Y", "Copy request as grpcurl"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+H", "Change gRPC host"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+W", "View parse warnings"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "F5", "Reconnect gRPC"))
	lines = append(lines, "")

	lines = append(lines, sectionStyle.Render("Explorer"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "j / Down", "Move cursor down"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "k / Up", "Move cursor up"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Enter / l", "Expand or select"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "h / Left", "Collapse or parent"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "g", "Jump to top"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "G", "Jump to bottom"))
	lines = append(lines, "")

	lines = append(lines, sectionStyle.Render("Request Builder (Normal Mode)"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "j / Down", "Next field"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "k / Up", "Previous field"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Enter", "Edit field / toggle section / enum / bool"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "a", "Add repeated/map entry"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "d", "Remove entry"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "any char", "Start typing to enter edit mode"))
	lines = append(lines, "")

	lines = append(lines, sectionStyle.Render("Request Builder (Edit Mode)"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Type", "Characters go into the field"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Esc", "Exit edit mode (back to normal)"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "Ctrl+Enter", "Send request"))
	lines = append(lines, "")

	lines = append(lines, sectionStyle.Render("Response Viewer"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "j / Down", "Scroll down"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "k / Up", "Scroll up"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "d / PgDn", "Half page down"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "u / PgUp", "Half page up"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "g", "Scroll to top"))
	lines = append(lines, fmtBinding(keyStyle, descStyle, "G", "Scroll to bottom"))
	lines = append(lines, "")

	lines = append(lines, DimmedStyle.Render("Press ? or Esc to close"))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 3).
		Render(content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// WarningsOverlay renders a modal showing parse warnings.
func WarningsOverlay(warnings []string, width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
	warnStyle := lipgloss.NewStyle().Foreground(ColorText)

	var lines []string
	lines = append(lines, titleStyle.Render("Parse Warnings"))
	lines = append(lines, "")

	if len(warnings) == 0 {
		lines = append(lines, DimmedStyle.Render("No warnings"))
	} else {
		for i, w := range warnings {
			// Truncate long warnings.
			maxLen := width - 20
			if maxLen < 40 {
				maxLen = 40
			}
			display := w
			if len(display) > maxLen {
				display = display[:maxLen-3] + "..."
			}
			lines = append(lines, warnStyle.Render(
				lipgloss.NewStyle().Bold(true).Foreground(ColorDimmed).Render(
					strings.Repeat(" ", 0)+string(rune('0'+i+1))+". ",
				)+display,
			))
		}
	}

	lines = append(lines, "")
	lines = append(lines, DimmedStyle.Render("Press Esc or Ctrl+W to close"))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorWarning).
		Padding(1, 3).
		Render(content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

func fmtBinding(keyStyle, descStyle lipgloss.Style, key, desc string) string {
	// Pad key to 16 chars for alignment.
	padded := key + strings.Repeat(" ", max(0, 16-len(key)))
	return "  " + keyStyle.Render(padded) + descStyle.Render(desc)
}
