package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// KeyBinding represents a keyboard shortcut for the help bar.
type KeyBinding struct {
	Key         string
	Description string
}

// HelpBar renders a context-sensitive shortcut legend at the bottom.
type HelpBar struct {
	width int
}

// NewHelpBar creates a new HelpBar.
func NewHelpBar() HelpBar {
	return HelpBar{}
}

// SetWidth sets the available width for the help bar.
func (h *HelpBar) SetWidth(width int) {
	h.width = width
}

// View renders the help bar with keybindings and optional warnings.
func (h HelpBar) View(bindings []KeyBinding, warnings []string) string {
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	descStyle := lipgloss.NewStyle().
		Foreground(ColorDimmed)

	sepStyle := lipgloss.NewStyle().
		Foreground(ColorBorder)

	warnStyle := lipgloss.NewStyle().
		Foreground(ColorWarning)

	sep := sepStyle.Render(" │ ")

	var parts []string
	for _, b := range bindings {
		parts = append(parts, keyStyle.Render(b.Key)+descStyle.Render(": "+b.Description))
	}
	left := strings.Join(parts, sep)

	var right string
	if len(warnings) > 0 {
		right = warnStyle.Render(strings.Join(warnings, " │ "))
	}

	if h.width <= 0 {
		if right != "" {
			return left + "  " + right
		}
		return left
	}

	style := HelpBarStyle.Width(h.width)
	if right == "" {
		return style.Render(left)
	}

	// Place warnings on the right side.
	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	gap := h.width - leftLen - rightLen
	if gap < 2 {
		gap = 2
	}
	return style.Render(left + strings.Repeat(" ", gap) + right)
}
