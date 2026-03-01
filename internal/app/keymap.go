package app

import "github.com/futuramacoder/protopilot/internal/ui"

// DefaultHelpBindings returns context-sensitive keybindings for the help bar.
func DefaultHelpBindings(focus PaneID, editing bool) []ui.KeyBinding {
	bindings := []ui.KeyBinding{
		{Key: "Tab", Description: "switch pane"},
	}

	switch focus {
	case PaneExplorer:
		bindings = append(bindings,
			ui.KeyBinding{Key: "j/k", Description: "navigate"},
			ui.KeyBinding{Key: "Enter", Description: "select"},
			ui.KeyBinding{Key: "h/l", Description: "collapse/expand"},
			ui.KeyBinding{Key: "?", Description: "help"},
			ui.KeyBinding{Key: "Ctrl+C", Description: "quit"},
		)

	case PaneRequestBuilder:
		if editing {
			bindings = append(bindings,
				ui.KeyBinding{Key: "Type", Description: "to edit"},
				ui.KeyBinding{Key: "Esc", Description: "stop editing"},
				ui.KeyBinding{Key: "Ctrl+Enter", Description: "send"},
			)
		} else {
			bindings = append(bindings,
				ui.KeyBinding{Key: "j/k", Description: "navigate"},
				ui.KeyBinding{Key: "Enter", Description: "edit/toggle"},
				ui.KeyBinding{Key: "a", Description: "add"},
				ui.KeyBinding{Key: "d", Description: "delete"},
				ui.KeyBinding{Key: "Ctrl+Enter", Description: "send"},
				ui.KeyBinding{Key: "?", Description: "help"},
			)
		}

	case PaneResponseViewer:
		bindings = append(bindings,
			ui.KeyBinding{Key: "j/k", Description: "scroll"},
			ui.KeyBinding{Key: "g/G", Description: "top/bottom"},
			ui.KeyBinding{Key: "?", Description: "help"},
			ui.KeyBinding{Key: "Ctrl+C", Description: "quit"},
		)
	}

	return bindings
}
