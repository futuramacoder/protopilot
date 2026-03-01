package ui

import (
	"charm.land/lipgloss/v2"
)

const asciiLogo = ` ___         _
| _ \_ _ ___| |_ ___
|  _/ '_/ _ \  _/ _ \
|_| |_| \___/\__\___/
    p i l o t`

const asciiLogoWidth = 22

// Logo returns the ASCII art logo string.
func Logo() string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(asciiLogo)
}

// EmptyState returns the logo + contextual hint message centered.
// When width is too narrow for the logo, only the hint is shown.
func EmptyState(hint string, width, height int) string {
	hintStyled := lipgloss.NewStyle().
		Foreground(ColorDimmed).
		Italic(true).
		Render(hint)

	var content string
	if width >= asciiLogoWidth+4 {
		content = Logo() + "\n\n" + hintStyled
	} else {
		content = hintStyled
	}

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}
