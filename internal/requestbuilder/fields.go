package requestbuilder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/ui"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FieldWidget is the interface for all input widgets.
type FieldWidget interface {
	View(focused bool) string
	Update(msg tea.Msg) (FieldWidget, tea.Cmd)
	Value() string
	SetValue(s string)
	Placeholder() string
	Validate() string
}

// newTextInput creates a styled textinput.Model with the given placeholder.
func newTextInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
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
	return ti
}

// viewTextInput renders a textinput with the correct focus state.
func viewTextInput(ti *textinput.Model, focused bool) string {
	if focused {
		if !ti.Focused() {
			ti.Focus()
		}
	} else {
		if ti.Focused() {
			ti.Blur()
		}
	}
	return ti.View()
}

// --- ScalarWidget ---

// ScalarWidget wraps bubbles/textinput for text and numeric inputs.
type ScalarWidget struct {
	input       textinput.Model
	placeholder string
	kind        protoreflect.Kind
}

func NewScalarWidget(kind protoreflect.Kind) *ScalarWidget {
	ph := scalarPlaceholder(kind)
	return &ScalarWidget{
		input:       newTextInput(ph),
		placeholder: ph,
		kind:        kind,
	}
}

func (w *ScalarWidget) View(focused bool) string {
	return viewTextInput(&w.input, focused)
}

func (w *ScalarWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	return w, cmd
}

func (w *ScalarWidget) Value() string       { return w.input.Value() }
func (w *ScalarWidget) SetValue(s string)   { w.input.SetValue(s) }
func (w *ScalarWidget) Placeholder() string { return w.placeholder }

func (w *ScalarWidget) Validate() string {
	v := w.input.Value()
	if v == "" {
		return ""
	}
	return validateScalar(v, w.kind)
}

func scalarPlaceholder(kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "bytes (base64)"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32, default: 0"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64, default: 0"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32, default: 0"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64, default: 0"
	case protoreflect.FloatKind:
		return "float, default: 0"
	case protoreflect.DoubleKind:
		return "double, default: 0"
	default:
		return "value"
	}
}

func validateScalar(v string, kind protoreflect.Kind) string {
	switch kind {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if _, err := strconv.ParseInt(v, 10, 32); err != nil {
			return "not a valid int32"
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			return "not a valid int64"
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if _, err := strconv.ParseUint(v, 10, 32); err != nil {
			return "not a valid uint32"
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if _, err := strconv.ParseUint(v, 10, 64); err != nil {
			return "not a valid uint64"
		}
	case protoreflect.FloatKind:
		if _, err := strconv.ParseFloat(v, 32); err != nil {
			return "not a valid float"
		}
	case protoreflect.DoubleKind:
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return "not a valid double"
		}
	}
	return ""
}

// --- BoolWidget ---

// BoolWidget displays a toggleable true/false value.
type BoolWidget struct {
	value bool
}

func NewBoolWidget() *BoolWidget {
	return &BoolWidget{value: false}
}

func (w *BoolWidget) View(focused bool) string {
	label := fmt.Sprintf("[%v]", w.value)
	if focused {
		return lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary).Render(label)
	}
	return lipgloss.NewStyle().Foreground(ui.ColorText).Render(label)
}

func (w *BoolWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter", " ":
			w.value = !w.value
		}
	}
	return w, nil
}

func (w *BoolWidget) Value() string {
	return strconv.FormatBool(w.value)
}

func (w *BoolWidget) SetValue(s string) {
	w.value, _ = strconv.ParseBool(s)
}

func (w *BoolWidget) Placeholder() string { return "bool, default: false" }
func (w *BoolWidget) Validate() string    { return "" }

// --- EnumWidget ---

// EnumWidget displays the current enum value. The enum popup is managed by the model.
type EnumWidget struct {
	current     string
	values      []string
	placeholder string
}

func NewEnumWidget(values []protoreflect.EnumValueDescriptor) *EnumWidget {
	names := make([]string, len(values))
	for i, v := range values {
		names[i] = string(v.Name())
	}
	current := ""
	if len(names) > 0 {
		current = names[0]
	}
	return &EnumWidget{
		current:     current,
		values:      names,
		placeholder: "enum",
	}
}

func (w *EnumWidget) View(focused bool) string {
	label := w.current
	if label == "" {
		label = ui.DimmedStyle.Render(w.placeholder)
	} else if focused {
		label = lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary).Render(label)
	} else {
		label = lipgloss.NewStyle().Foreground(ui.ColorText).Render(label)
	}
	return label
}

func (w *EnumWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	return w, nil
}

func (w *EnumWidget) Value() string       { return w.current }
func (w *EnumWidget) SetValue(s string)   { w.current = s }
func (w *EnumWidget) Placeholder() string { return w.placeholder }
func (w *EnumWidget) Values() []string    { return w.values }

func (w *EnumWidget) Validate() string {
	if w.current == "" {
		return ""
	}
	for _, v := range w.values {
		if v == w.current {
			return ""
		}
	}
	return "not a valid enum value"
}

// --- TimestampWidget ---

// TimestampWidget is a text input expecting RFC3339 format.
type TimestampWidget struct {
	input textinput.Model
}

func NewTimestampWidget() *TimestampWidget {
	return &TimestampWidget{input: newTextInput("2006-01-02T15:04:05Z")}
}

func (w *TimestampWidget) View(focused bool) string {
	return viewTextInput(&w.input, focused)
}

func (w *TimestampWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	return w, cmd
}

func (w *TimestampWidget) Value() string       { return w.input.Value() }
func (w *TimestampWidget) SetValue(s string)   { w.input.SetValue(s) }
func (w *TimestampWidget) Placeholder() string { return "2006-01-02T15:04:05Z" }

func (w *TimestampWidget) Validate() string {
	v := w.input.Value()
	if v == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return "not a valid RFC3339 timestamp"
	}
	return ""
}

// --- DurationWidget ---

// DurationWidget is a text input expecting Go duration format.
type DurationWidget struct {
	input textinput.Model
}

func NewDurationWidget() *DurationWidget {
	return &DurationWidget{input: newTextInput("e.g., 5s, 1m30s, 500ms")}
}

func (w *DurationWidget) View(focused bool) string {
	return viewTextInput(&w.input, focused)
}

func (w *DurationWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	return w, cmd
}

func (w *DurationWidget) Value() string       { return w.input.Value() }
func (w *DurationWidget) SetValue(s string)   { w.input.SetValue(s) }
func (w *DurationWidget) Placeholder() string { return "e.g., 5s, 1m30s, 500ms" }

func (w *DurationWidget) Validate() string {
	v := w.input.Value()
	if v == "" {
		return ""
	}
	if _, err := time.ParseDuration(v); err != nil {
		return "not a valid duration"
	}
	return ""
}

// --- StructWidget ---

// StructWidget is a text input for raw JSON (Struct/Value types).
type StructWidget struct {
	input textinput.Model
}

func NewStructWidget() *StructWidget {
	return &StructWidget{input: newTextInput(`{"key": "value"}`)}
}

func (w *StructWidget) View(focused bool) string {
	return viewTextInput(&w.input, focused)
}

func (w *StructWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	return w, cmd
}

func (w *StructWidget) Value() string       { return w.input.Value() }
func (w *StructWidget) SetValue(s string)   { w.input.SetValue(s) }
func (w *StructWidget) Placeholder() string { return `{"key": "value"}` }

func (w *StructWidget) Validate() string {
	v := w.input.Value()
	if v == "" {
		return ""
	}
	if !json.Valid([]byte(v)) {
		return "not valid JSON"
	}
	return ""
}

// --- WrapperWidget ---

// WrapperWidget is a nullable scalar input for wrapper types.
type WrapperWidget struct {
	input  textinput.Model
	isNull bool
	inner  protoreflect.Kind
}

func NewWrapperWidget(innerKind protoreflect.Kind) *WrapperWidget {
	ph := scalarPlaceholder(innerKind) + " (nullable)"
	return &WrapperWidget{
		input:  newTextInput(ph),
		isNull: true,
		inner:  innerKind,
	}
}

func (w *WrapperWidget) View(focused bool) string {
	if w.isNull {
		label := "null"
		if focused {
			return lipgloss.NewStyle().Bold(true).Foreground(ui.ColorDimmed).Render(label)
		}
		return ui.DimmedStyle.Render(label)
	}
	return viewTextInput(&w.input, focused)
}

func (w *WrapperWidget) Update(msg tea.Msg) (FieldWidget, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "ctrl+n" {
			w.isNull = !w.isNull
			return w, nil
		}
	}
	if !w.isNull {
		var cmd tea.Cmd
		w.input, cmd = w.input.Update(msg)
		return w, cmd
	}
	return w, nil
}

func (w *WrapperWidget) Value() string {
	if w.isNull {
		return ""
	}
	return w.input.Value()
}

func (w *WrapperWidget) SetValue(s string) {
	if s == "" {
		w.isNull = true
		return
	}
	w.isNull = false
	w.input.SetValue(s)
}

func (w *WrapperWidget) Placeholder() string {
	return w.input.Placeholder
}

func (w *WrapperWidget) Validate() string {
	if w.isNull {
		return ""
	}
	v := w.input.Value()
	if v == "" {
		return ""
	}
	return validateScalar(v, w.inner)
}

func (w *WrapperWidget) IsNull() bool { return w.isNull }

// wrapperInnerKind returns the proto Kind for the "value" field of a wrapper type.
func wrapperInnerKind(fullName string) protoreflect.Kind {
	switch {
	case strings.HasSuffix(fullName, "StringValue"):
		return protoreflect.StringKind
	case strings.HasSuffix(fullName, "BytesValue"):
		return protoreflect.BytesKind
	case strings.HasSuffix(fullName, "BoolValue"):
		return protoreflect.BoolKind
	case strings.HasSuffix(fullName, "Int32Value"):
		return protoreflect.Int32Kind
	case strings.HasSuffix(fullName, "Int64Value"):
		return protoreflect.Int64Kind
	case strings.HasSuffix(fullName, "UInt32Value"):
		return protoreflect.Uint32Kind
	case strings.HasSuffix(fullName, "UInt64Value"):
		return protoreflect.Uint64Kind
	case strings.HasSuffix(fullName, "FloatValue"):
		return protoreflect.FloatKind
	case strings.HasSuffix(fullName, "DoubleValue"):
		return protoreflect.DoubleKind
	default:
		return protoreflect.StringKind
	}
}
