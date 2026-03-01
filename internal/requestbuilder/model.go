package requestbuilder

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"

	"github.com/futuramacoder/protopilot/internal/messages"
	"github.com/futuramacoder/protopilot/internal/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Model is the Bubble Tea model for the request builder pane.
type Model struct {
	method     protoreflect.MethodDescriptor
	metadata   MetadataSection
	fields     []FormField
	flatFields []*FormField // flattened visible fields for navigation
	focusIdx   int
	inMetadata bool       // true if focus is in metadata section
	editing    bool       // true when in edit/insert mode for text fields
	enumPopup  *EnumPopup // non-nil when enum overlay is open
	focused    bool
	width      int
	height     int
	viewport   viewport.Model
}

// EnumPopup is the overlay for selecting enum values.
type EnumPopup struct {
	Values   []string
	Cursor   int
	FieldIdx int // which field this popup is for
}

// New creates a new empty request builder model.
func New() Model {
	return Model{
		metadata: NewMetadataSection(),
		viewport: viewport.New(),
	}
}

// SetMethod configures the builder for a new RPC method.
func (m *Model) SetMethod(md protoreflect.MethodDescriptor) {
	m.method = md
	m.fields = Generate(md.Input())
	m.focusIdx = 0
	m.inMetadata = false
	m.editing = false
	m.enumPopup = nil
	m.flatFields = flattenFormFields(m.fields)
}

// Method returns the current method descriptor.
func (m *Model) Method() protoreflect.MethodDescriptor {
	return m.method
}

// Editing returns true when the request builder is in edit/insert mode.
func (m *Model) Editing() bool {
	return m.editing
}

// Fields returns the form fields.
func (m *Model) Fields() []FormField {
	return m.fields
}

// Metadata returns the metadata section.
func (m *Model) MetadataMap() map[string]string {
	return m.metadata.ToMap()
}

// CollectValues gathers all field values as map[string]any for the codec.
func (m *Model) CollectValues() map[string]any {
	return collectValues(m.fields)
}

func collectValues(fields []FormField) map[string]any {
	result := make(map[string]any)
	for _, f := range fields {
		// Skip inactive oneof variants.
		if f.OneofGroup != "" && !f.OneofActive {
			continue
		}

		switch f.Info.Kind {
		case proto.FieldKindOneof:
			// Recurse into oneof children.
			for k, v := range collectValues(f.Children) {
				result[k] = v
			}
		case proto.FieldKindMessage:
			if len(f.Children) > 0 {
				result[f.Info.Name] = collectValues(f.Children)
			}
		case proto.FieldKindRepeated:
			var items []any
			for _, child := range f.Children {
				if child.Widget != nil {
					if v := child.Widget.Value(); v != "" {
						items = append(items, v)
					}
				} else if len(child.Children) > 0 {
					items = append(items, collectValues(child.Children))
				}
			}
			if len(items) > 0 {
				result[f.Info.Name] = items
			}
		case proto.FieldKindMap:
			mapVals := make(map[string]any)
			for _, entry := range f.Children {
				if len(entry.Children) >= 2 {
					k := entry.Children[0].Widget.Value()
					v := entry.Children[1].Widget.Value()
					if k != "" {
						mapVals[k] = v
					}
				}
			}
			if len(mapVals) > 0 {
				result[f.Info.Name] = mapVals
			}
		default:
			if f.Widget != nil {
				if v := f.Widget.Value(); v != "" {
					result[f.Info.Name] = v
				}
			}
		}
	}
	return result
}

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	// Handle enum popup first — it captures all keys.
	if m.enumPopup != nil {
		return m.handleEnumPopup(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Delegate to focused widget for non-key messages.
	return m.updateFocusedWidget(msg)
}

func (m Model) handleEnumPopup(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "j", "down":
		if m.enumPopup.Cursor < len(m.enumPopup.Values)-1 {
			m.enumPopup.Cursor++
		}
	case "k", "up":
		if m.enumPopup.Cursor > 0 {
			m.enumPopup.Cursor--
		}
	case "enter":
		// Apply selection.
		if m.enumPopup.FieldIdx >= 0 && m.enumPopup.FieldIdx < len(m.flatFields) {
			f := m.flatFields[m.enumPopup.FieldIdx]
			if f.Widget != nil {
				f.Widget.SetValue(m.enumPopup.Values[m.enumPopup.Cursor])
			}
		}
		m.enumPopup = nil
	case "escape", "esc":
		m.enumPopup = nil
	}

	return m, nil
}

// isEditableField returns true if the focused field supports text editing.
func (m Model) isEditableField() bool {
	if len(m.flatFields) == 0 || m.focusIdx >= len(m.flatFields) {
		return false
	}
	f := m.flatFields[m.focusIdx]
	switch f.Info.Kind {
	case proto.FieldKindScalar, proto.FieldKindTimestamp, proto.FieldKindDuration,
		proto.FieldKindStruct, proto.FieldKindWrapper:
		return f.Widget != nil
	}
	return false
}

// isPrintable returns true if the key event represents a printable character.
func isPrintable(msg tea.KeyPressMsg) bool {
	return msg.Text != "" && msg.Code >= 32
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	// Edit mode: forward everything to the widget except Esc and Ctrl+Enter.
	if m.editing {
		switch s {
		case "escape", "esc":
			m.editing = false
			return m, nil
		case "ctrl+enter":
			m.editing = false
			return m.handleSubmit()
		default:
			return m.updateFocusedWidget(msg)
		}
	}

	// Normal mode below.

	// Toggle metadata section focus.
	if m.inMetadata {
		switch s {
		case "j", "down":
			m.inMetadata = false
			m.focusIdx = 0
			return m, nil
		case "k", "up":
			return m, nil // Already at top.
		}
		cmd := m.metadata.Update(msg)
		return m, cmd
	}

	switch s {
	case "k", "up":
		if m.focusIdx > 0 {
			m.focusIdx--
		} else {
			// Move to metadata section.
			m.inMetadata = true
		}
	case "j", "down":
		if m.focusIdx < len(m.flatFields)-1 {
			m.focusIdx++
		}
	case "enter":
		return m.handleEnter()
	case "a":
		return m.handleAdd()
	case "d":
		return m.handleDelete()
	case "ctrl+enter":
		return m.handleSubmit()
	default:
		// Auto-enter edit mode on printable character when on an editable field.
		if isPrintable(msg) && m.isEditableField() {
			m.editing = true
			return m.updateFocusedWidget(msg)
		}
		return m.updateFocusedWidget(msg)
	}

	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if len(m.flatFields) == 0 || m.focusIdx >= len(m.flatFields) {
		return m, nil
	}

	f := m.flatFields[m.focusIdx]

	switch f.Info.Kind {
	case proto.FieldKindMessage, proto.FieldKindRepeated, proto.FieldKindMap:
		f.Expanded = !f.Expanded
		m.flatFields = flattenFormFields(m.fields)
		// Clamp focus.
		if m.focusIdx >= len(m.flatFields) {
			m.focusIdx = len(m.flatFields) - 1
		}

	case proto.FieldKindBool:
		if f.Widget != nil {
			f.Widget.Update(msg(tea.KeyPressMsg(tea.Key{Code: rune(tea.KeyEnter), Text: "enter"})))
		}

	case proto.FieldKindEnum:
		if ew, ok := f.Widget.(*EnumWidget); ok {
			m.enumPopup = &EnumPopup{
				Values:   ew.Values(),
				Cursor:   0,
				FieldIdx: m.focusIdx,
			}
		}

	case proto.FieldKindOneof:
		// This is a oneof group header — do nothing, handled through children.

	default:
		// For oneof variant children, activate this variant.
		if f.OneofGroup != "" {
			m.activateOneofVariant(f)
			return m, nil
		}
		// Enter edit mode for editable text fields.
		if f.Widget != nil {
			m.editing = true
		}
	}

	return m, nil
}

func (m *Model) activateOneofVariant(target *FormField) {
	// Find all siblings in the same oneof group and toggle.
	for i := range m.fields {
		if m.fields[i].Info.Kind == proto.FieldKindOneof {
			for j := range m.fields[i].Children {
				child := &m.fields[i].Children[j]
				child.OneofActive = child.Info.Name == target.Info.Name
			}
		}
	}
	m.flatFields = flattenFormFields(m.fields)
}

func (m Model) handleAdd() (tea.Model, tea.Cmd) {
	if len(m.flatFields) == 0 || m.focusIdx >= len(m.flatFields) {
		return m, nil
	}
	f := m.flatFields[m.focusIdx]
	switch f.Info.Kind {
	case proto.FieldKindRepeated:
		AddRepeatedEntry(f)
		f.Expanded = true
		m.flatFields = flattenFormFields(m.fields)
	case proto.FieldKindMap:
		AddMapEntry(f)
		f.Expanded = true
		m.flatFields = flattenFormFields(m.fields)
	}
	return m, nil
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	// Find the parent repeated/map field and remove the focused child.
	if len(m.flatFields) == 0 || m.focusIdx >= len(m.flatFields) {
		return m, nil
	}
	f := m.flatFields[m.focusIdx]
	parent := findParentField(m.fields, f)
	if parent != nil && (parent.Info.Kind == proto.FieldKindRepeated || parent.Info.Kind == proto.FieldKindMap) {
		for i := range parent.Children {
			if &parent.Children[i] == f {
				RemoveEntry(parent, i)
				m.flatFields = flattenFormFields(m.fields)
				if m.focusIdx >= len(m.flatFields) && len(m.flatFields) > 0 {
					m.focusIdx = len(m.flatFields) - 1
				}
				break
			}
		}
	}
	return m, nil
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	// Validate all fields.
	errs := ValidateAll(m.fields)
	if len(errs) > 0 {
		// Mark validation errors on fields.
		applyValidationErrors(m.fields, errs)
		return m, nil
	}

	if m.method == nil {
		return m, nil
	}

	return m, func() tea.Msg {
		return messages.SendRequestMsg{
			MethodDesc:  m.method,
			FieldValues: m.CollectValues(),
			Metadata:    m.metadata.ToMap(),
		}
	}
}

func (m Model) updateFocusedWidget(tmsg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.flatFields) == 0 || m.focusIdx >= len(m.flatFields) {
		return m, nil
	}
	f := m.flatFields[m.focusIdx]
	if f.Widget == nil {
		return m, nil
	}

	w, cmd := f.Widget.Update(tmsg)
	f.Widget = w
	f.ValidationErr = ValidateField(f)
	return m, cmd
}

// applyValidationErrors marks validation errors on the matching fields.
func applyValidationErrors(fields []FormField, errs []ValidationError) {
	errMap := make(map[string]string)
	for _, e := range errs {
		errMap[e.Path] = e.Message
	}
	applyErrRecursive(fields, errMap)
}

func applyErrRecursive(fields []FormField, errMap map[string]string) {
	for i := range fields {
		if msg, ok := errMap[fields[i].Info.Path]; ok {
			fields[i].ValidationErr = msg
		}
		if len(fields[i].Children) > 0 {
			applyErrRecursive(fields[i].Children, errMap)
		}
	}
}

// flattenFormFields returns a flat list of pointers to visible fields for navigation.
func flattenFormFields(fields []FormField) []*FormField {
	var result []*FormField
	flattenFF(fields, &result)
	return result
}

func flattenFF(fields []FormField, result *[]*FormField) {
	for i := range fields {
		f := &fields[i]

		if f.Info.Kind == proto.FieldKindOneof {
			*result = append(*result, f)
			for j := range f.Children {
				*result = append(*result, &f.Children[j])
			}
			continue
		}

		*result = append(*result, f)

		if f.Expanded && len(f.Children) > 0 {
			flattenFF(f.Children, result)
		}
	}
}

// findParentField finds the parent FormField that contains the target as a child.
func findParentField(fields []FormField, target *FormField) *FormField {
	for i := range fields {
		for j := range fields[i].Children {
			if &fields[i].Children[j] == target {
				return &fields[i]
			}
			if p := findParentField(fields[i].Children, target); p != nil {
				return p
			}
		}
	}
	return nil
}

// msg is a helper type to create a tea.Msg-compatible value.
type msg tea.KeyPressMsg
