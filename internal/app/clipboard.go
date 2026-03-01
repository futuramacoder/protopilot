package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

// ClipboardResultMsg is sent after a clipboard write attempt.
type ClipboardResultMsg struct {
	Success bool
	Err     error
}

// copyToClipboard returns a tea.Cmd that writes text to the system clipboard.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(text)
		return ClipboardResultMsg{
			Success: err == nil,
			Err:     err,
		}
	}
}
