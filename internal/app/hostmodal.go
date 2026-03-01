package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/ui"
)

// HostModal is a modal text input for changing the gRPC host.
type HostModal struct {
	input   textinput.Model
	visible bool
}

// NewHostModal creates a new host change modal.
func NewHostModal() HostModal {
	ti := textinput.New()
	ti.Placeholder = "host:port"
	ti.SetStyles(textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(ui.ColorText),
			Placeholder: lipgloss.NewStyle().Foreground(ui.ColorDimmed),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(ui.ColorText),
			Placeholder: lipgloss.NewStyle().Foreground(ui.ColorDimmed),
		},
	})
	return HostModal{input: ti}
}

// Show opens the modal, pre-filled with the current host.
func (h *HostModal) Show(currentHost string) tea.Cmd {
	h.visible = true
	h.input.SetValue(currentHost)
	return h.input.Focus()
}

// Hide closes the modal.
func (h *HostModal) Hide() {
	h.visible = false
	h.input.Blur()
}

// Visible returns whether the modal is showing.
func (h *HostModal) Visible() bool {
	return h.visible
}

// Value returns the current input value.
func (h *HostModal) Value() string {
	return strings.TrimSpace(h.input.Value())
}

// Update handles key events for the modal.
func (h *HostModal) Update(msg tea.Msg) (confirmed bool, cmd tea.Cmd) {
	if !h.visible {
		return false, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		h.input, cmd = h.input.Update(msg)
		return false, cmd
	}

	switch key.String() {
	case "enter":
		h.Hide()
		return true, nil
	case "escape", "esc":
		h.Hide()
		return false, nil
	default:
		h.input, cmd = h.input.Update(msg)
		return false, cmd
	}
}

// View renders the modal overlay.
func (h *HostModal) View(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	hintStyle := lipgloss.NewStyle().Foreground(ui.ColorDimmed)

	var lines []string
	lines = append(lines, titleStyle.Render("Change gRPC Host"))
	lines = append(lines, "")
	lines = append(lines, h.input.View())
	lines = append(lines, "")
	lines = append(lines, hintStyle.Render("Enter to confirm, Esc to cancel"))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 3).
		Width(50).
		Render(content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}
