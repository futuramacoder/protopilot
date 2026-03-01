package app

import "github.com/futuramacoder/protopilot/internal/ui"

// DefaultHelpBindings returns the keybindings shown in the help bar.
func DefaultHelpBindings() []ui.KeyBinding {
	return []ui.KeyBinding{
		{Key: "Tab", Description: "switch pane"},
		{Key: "Ctrl+Enter", Description: "send"},
		{Key: "?", Description: "help"},
		{Key: "Ctrl+C", Description: "quit"},
	}
}
