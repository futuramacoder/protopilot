package ui

import "charm.land/lipgloss/v2"

// Pre-built Lip Gloss styles used across all panes.
var (
	// PaneBorder is the style for unfocused pane borders.
	PaneBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	// PaneFocusedBorder is the style for focused pane borders (purple).
	PaneFocusedBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorFocusBorder)

	// TitleStyle is used for pane title text.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	// DimmedStyle is used for grayed text (placeholders, streaming methods).
	DimmedStyle = lipgloss.NewStyle().
			Foreground(ColorDimmed)

	// ErrorStyle is used for red text (validation errors).
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// SuccessStyle is used for green text (OK status).
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// WarningStyle is used for yellow text.
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// SelectedStyle is used for highlighted/selected items.
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary)

	// HelpBarStyle is used for the bottom help bar.
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(ColorDimmed).
			Background(ColorBackground)

	// SecondaryStyle is used for cyan highlight text.
	SecondaryStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)
)
