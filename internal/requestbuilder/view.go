package requestbuilder

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/proto"
	"github.com/futuramacoder/protopilot/internal/ui"
)

// View implements tea.Model.
func (m Model) View() tea.View {
	if m.method == nil {
		content := ui.EmptyState("Select a method from the explorer to begin", m.width-4, m.height-4)
		return tea.NewView(m.applyBorder(content))
	}

	var sections []string

	// Metadata section.
	sections = append(sections, m.metadata.View(m.focused && m.inMetadata))

	// Divider.
	sections = append(sections, ui.DimmedStyle.Render(strings.Repeat("─", m.width-4)))

	// Fields.
	fieldLines := m.renderFields()
	sections = append(sections, fieldLines...)

	content := strings.Join(sections, "\n")

	// Overlay enum popup if active.
	if m.enumPopup != nil {
		content = m.overlayEnumPopup(content)
	}

	return tea.NewView(m.applyBorder(content))
}

func (m Model) renderFields() []string {
	var lines []string
	flatIdx := 0
	renderFieldList(m.fields, &lines, &flatIdx, m.focusIdx, m.focused, 0)
	return lines
}

func renderFieldList(fields []FormField, lines *[]string, flatIdx *int, focusIdx int, paneIsFocused bool, depth int) {
	indent := strings.Repeat("  ", depth)

	for i := range fields {
		f := &fields[i]
		isFocused := paneIsFocused && *flatIdx == focusIdx
		currentIdx := *flatIdx
		_ = currentIdx

		switch f.Info.Kind {
		case proto.FieldKindOneof:
			// Render oneof group header.
			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
			*lines = append(*lines, indent+headerStyle.Render(f.Info.Name+":"))
			*flatIdx++

			// Render variants.
			for j := range f.Children {
				child := &f.Children[j]
				childFocused := paneIsFocused && *flatIdx == focusIdx

				bullet := "○"
				if child.OneofActive {
					bullet = "◉"
				}

				var line string
				if child.Widget != nil && child.OneofActive {
					line = fmt.Sprintf("%s  %s %s: %s", indent, bullet, child.Info.Name, child.Widget.View(childFocused))
				} else {
					nameStyle := ui.DimmedStyle
					if child.OneofActive {
						nameStyle = lipgloss.NewStyle().Foreground(ui.ColorText)
					}
					line = fmt.Sprintf("%s  %s %s", indent, bullet, nameStyle.Render(child.Info.Name))
				}
				*lines = append(*lines, line)

				if child.ValidationErr != "" {
					*lines = append(*lines, indent+"    "+ui.ErrorStyle.Render(child.ValidationErr))
				}
				*flatIdx++
			}

		case proto.FieldKindMessage:
			arrow := "▸"
			if f.Expanded {
				arrow = "▾"
			}
			headerStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
			if isFocused {
				headerStyle = headerStyle.Bold(true).Foreground(ui.ColorSecondary)
			}
			*lines = append(*lines, indent+headerStyle.Render(fmt.Sprintf("%s %s", arrow, f.Info.Name)))
			*flatIdx++

			if f.Expanded {
				renderFieldList(f.Children, lines, flatIdx, focusIdx, paneIsFocused, depth+1)
			}

		case proto.FieldKindRepeated:
			arrow := "▸"
			if f.Expanded {
				arrow = "▾"
			}
			headerStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
			if isFocused {
				headerStyle = headerStyle.Bold(true).Foreground(ui.ColorSecondary)
			}
			*lines = append(*lines, indent+headerStyle.Render(fmt.Sprintf("%s %s (%d)", arrow, f.Info.Name, len(f.Children))))
			*flatIdx++

			if f.Expanded {
				for j := range f.Children {
					child := &f.Children[j]
					childFocused := paneIsFocused && *flatIdx == focusIdx
					if child.Widget != nil {
						line := fmt.Sprintf("%s  [%d]: %s", indent, j, child.Widget.View(childFocused))
						*lines = append(*lines, line)
					} else {
						*lines = append(*lines, indent+fmt.Sprintf("  [%d]:", j))
						*flatIdx++
						renderFieldList(child.Children, lines, flatIdx, focusIdx, paneIsFocused, depth+2)
						continue
					}
					if child.ValidationErr != "" {
						*lines = append(*lines, indent+"    "+ui.ErrorStyle.Render(child.ValidationErr))
					}
					*flatIdx++
				}
			}

		case proto.FieldKindMap:
			arrow := "▸"
			if f.Expanded {
				arrow = "▾"
			}
			headerStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
			if isFocused {
				headerStyle = headerStyle.Bold(true).Foreground(ui.ColorSecondary)
			}
			*lines = append(*lines, indent+headerStyle.Render(fmt.Sprintf("%s %s (%d)", arrow, f.Info.Name, len(f.Children))))
			*flatIdx++

			if f.Expanded {
				for j := range f.Children {
					entry := &f.Children[j]
					childFocused := paneIsFocused && *flatIdx == focusIdx
					if len(entry.Children) >= 2 {
						keyView := entry.Children[0].Widget.View(childFocused)
						valView := entry.Children[1].Widget.View(childFocused)
						*lines = append(*lines, fmt.Sprintf("%s  %s → %s", indent, keyView, valView))
					}
					*flatIdx++
				}
			}

		default:
			// Scalar, Bool, Enum, Timestamp, Duration, Struct, Wrapper.
			if f.Widget != nil {
				label := f.Info.Name
				nameStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
				if isFocused {
					nameStyle = nameStyle.Bold(true)
				}
				line := fmt.Sprintf("%s%s: %s", indent, nameStyle.Render(label), f.Widget.View(isFocused))
				*lines = append(*lines, line)
			}
			if f.ValidationErr != "" {
				*lines = append(*lines, indent+"  "+ui.ErrorStyle.Render(f.ValidationErr))
			}
			*flatIdx++
		}
	}
}

func (m Model) overlayEnumPopup(content string) string {
	if m.enumPopup == nil {
		return content
	}

	var popupLines []string
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(0, 1)

	for i, val := range m.enumPopup.Values {
		if i == m.enumPopup.Cursor {
			popupLines = append(popupLines, ui.SelectedStyle.Render(val))
		} else {
			popupLines = append(popupLines, lipgloss.NewStyle().Foreground(ui.ColorText).Render(val))
		}
	}

	popup := borderStyle.Render(strings.Join(popupLines, "\n"))

	// Simple overlay: append popup below content.
	return content + "\n" + popup
}

func (m Model) applyBorder(content string) string {
	border := ui.PaneBorder
	if m.focused {
		border = ui.PaneFocusedBorder
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	title := titleStyle.Render(" Request Builder ")

	if m.focused && m.method != nil {
		modeStyle := lipgloss.NewStyle().Bold(true)
		if m.editing {
			modeStyle = modeStyle.Foreground(ui.ColorWarning)
			title += " " + modeStyle.Render("[INSERT]")
		} else {
			modeStyle = modeStyle.Foreground(ui.ColorDimmed)
			title += " " + modeStyle.Render("[NORMAL]")
		}
	}

	return border.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(title + "\n" + content)
}
