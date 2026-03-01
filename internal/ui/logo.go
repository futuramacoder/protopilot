package ui

import (
	"charm.land/lipgloss/v2"
)

const asciiLogo = `
 ____            _              _ _       _
|  _ \ _ __ ___ | |_ ___  _ __ (_) | ___ | |_
| |_) | '__/ _ \| __/ _ \| '_ \| | |/ _ \| __|
|  __/| | | (_) | || (_) | |_) | | | (_) | |_
|_|   |_|  \___/ \__\___/| .__/|_|_|\___/ \__|
                          |_|                  `

// Logo returns the ASCII art logo string.
func Logo() string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(asciiLogo)
}

// EmptyState returns the logo + contextual hint message centered.
func EmptyState(hint string, width, height int) string {
	logo := Logo()
	hintStyled := lipgloss.NewStyle().
		Foreground(ColorDimmed).
		Italic(true).
		Render(hint)

	content := logo + "\n\n" + hintStyled

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}
