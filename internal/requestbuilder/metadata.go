package requestbuilder

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/ui"
)

// MetadataSection manages the collapsible metadata key-value editor.
type MetadataSection struct {
	Entries  []MetadataEntry
	Expanded bool
	FocusIdx int
	FocusCol int // 0=key, 1=value
}

// MetadataEntry is a single key-value pair.
type MetadataEntry struct {
	Key   textinput.Model
	Value textinput.Model
}

// NewMetadataSection creates a new empty metadata section.
func NewMetadataSection() MetadataSection {
	return MetadataSection{
		Expanded: false,
	}
}

func newMetadataEntry() MetadataEntry {
	return MetadataEntry{
		Key:   newTextInput("key"),
		Value: newTextInput("value"),
	}
}

// AddEntry appends a new empty key-value row.
func (m *MetadataSection) AddEntry() {
	m.Entries = append(m.Entries, newMetadataEntry())
	m.FocusIdx = len(m.Entries) - 1
	m.FocusCol = 0
}

// RemoveEntry removes the entry at the given index.
func (m *MetadataSection) RemoveEntry(idx int) {
	if idx < 0 || idx >= len(m.Entries) {
		return
	}
	m.Entries = append(m.Entries[:idx], m.Entries[idx+1:]...)
	if m.FocusIdx >= len(m.Entries) && len(m.Entries) > 0 {
		m.FocusIdx = len(m.Entries) - 1
	}
}

// ToMap returns the metadata as map[string]string for gRPC headers.
func (m *MetadataSection) ToMap() map[string]string {
	result := make(map[string]string)
	for _, e := range m.Entries {
		k := strings.TrimSpace(e.Key.Value())
		v := strings.TrimSpace(e.Value.Value())
		if k != "" {
			result[k] = v
		}
	}
	return result
}

// Update handles key events when the metadata section is focused.
func (m *MetadataSection) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m.updateActiveInput(msg)
	}

	switch key.String() {
	case "enter":
		m.Expanded = !m.Expanded
		return nil
	case "a":
		if m.Expanded {
			m.AddEntry()
		}
		return nil
	case "d":
		if m.Expanded && len(m.Entries) > 0 {
			m.RemoveEntry(m.FocusIdx)
		}
		return nil
	case "tab":
		if m.Expanded && len(m.Entries) > 0 {
			if m.FocusCol == 0 {
				m.FocusCol = 1
			} else {
				m.FocusCol = 0
				m.FocusIdx++
				if m.FocusIdx >= len(m.Entries) {
					m.FocusIdx = 0
				}
			}
		}
		return nil
	case "shift+tab":
		if m.Expanded && len(m.Entries) > 0 {
			if m.FocusCol == 1 {
				m.FocusCol = 0
			} else {
				m.FocusCol = 1
				m.FocusIdx--
				if m.FocusIdx < 0 {
					m.FocusIdx = len(m.Entries) - 1
				}
			}
		}
		return nil
	}

	return m.updateActiveInput(msg)
}

func (m *MetadataSection) updateActiveInput(msg tea.Msg) tea.Cmd {
	if !m.Expanded || len(m.Entries) == 0 {
		return nil
	}
	if m.FocusIdx < 0 || m.FocusIdx >= len(m.Entries) {
		return nil
	}

	var cmd tea.Cmd
	if m.FocusCol == 0 {
		m.Entries[m.FocusIdx].Key, cmd = m.Entries[m.FocusIdx].Key.Update(msg)
	} else {
		m.Entries[m.FocusIdx].Value, cmd = m.Entries[m.FocusIdx].Value.Update(msg)
	}
	return cmd
}

// View renders the metadata section.
func (m *MetadataSection) View(focused bool) string {
	arrow := "▸"
	if m.Expanded {
		arrow = "▾"
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	header := headerStyle.Render(fmt.Sprintf("%s Metadata (%d)", arrow, len(m.Entries)))

	if !m.Expanded {
		return header
	}

	var lines []string
	lines = append(lines, header)

	for i := range m.Entries {
		keyFocused := focused && i == m.FocusIdx && m.FocusCol == 0
		valFocused := focused && i == m.FocusIdx && m.FocusCol == 1

		keyView := viewTextInput(&m.Entries[i].Key, keyFocused)
		valView := viewTextInput(&m.Entries[i].Value, valFocused)

		line := fmt.Sprintf("  %s: %s", keyView, valView)
		lines = append(lines, line)
	}

	addStyle := ui.DimmedStyle
	lines = append(lines, addStyle.Render("  + Add (a) / Remove (d)"))

	return strings.Join(lines, "\n")
}
