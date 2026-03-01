package requestbuilder

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/futuramacoder/protopilot/internal/proto"
)

// FormField represents a single field in the request form.
type FormField struct {
	Info          proto.FieldInfo
	Widget        FieldWidget // The input widget (interface)
	Value         any         // Current value (nil = not set)
	Expanded      bool        // For collapsible sections
	ValidationErr string      // Current validation error (empty = valid)
	Children      []FormField // For message/repeated/map fields
	OneofGroup    string      // Oneof group name (empty if not in oneof)
	OneofActive   bool        // Whether this variant is selected
}

// Generate creates a list of FormField entries from a method's input
// message descriptor. Each field is classified by FieldKind and gets
// the appropriate widget configuration.
func Generate(md protoreflect.MessageDescriptor) []FormField {
	infos := proto.WalkMessage(md)
	return generateFields(infos)
}

func generateFields(infos []proto.FieldInfo) []FormField {
	var fields []FormField

	for _, info := range infos {
		field := FormField{
			Info: info,
		}

		switch info.Kind {
		case proto.FieldKindScalar:
			field.Widget = NewScalarWidget(info.Desc.Kind())

		case proto.FieldKindBool:
			field.Widget = NewBoolWidget()

		case proto.FieldKindEnum:
			field.Widget = NewEnumWidget(info.EnumValues)

		case proto.FieldKindTimestamp:
			field.Widget = NewTimestampWidget()

		case proto.FieldKindDuration:
			field.Widget = NewDurationWidget()

		case proto.FieldKindStruct:
			field.Widget = NewStructWidget()

		case proto.FieldKindWrapper:
			innerKind := protoreflect.StringKind
			if info.Desc != nil && info.Desc.Message() != nil {
				innerKind = wrapperInnerKind(string(info.Desc.Message().FullName()))
			}
			field.Widget = NewWrapperWidget(innerKind)

		case proto.FieldKindMessage:
			field.Expanded = false
			field.Children = generateFields(info.Children)

		case proto.FieldKindRepeated:
			field.Expanded = false
			// Children will be added dynamically via AddEntry.

		case proto.FieldKindMap:
			field.Expanded = false
			// Children will be added dynamically via AddEntry.

		case proto.FieldKindOneof:
			field.Expanded = true
			// Generate children for each oneof variant.
			for i, peer := range info.OneofPeers {
				child := FormField{
					Info:        peer,
					OneofGroup:  info.Name,
					OneofActive: i == 0, // First variant active by default.
				}
				child.Widget = createWidgetForField(peer)
				field.Children = append(field.Children, child)
			}
		}

		fields = append(fields, field)
	}

	return fields
}

// createWidgetForField creates the appropriate widget for a FieldInfo.
func createWidgetForField(info proto.FieldInfo) FieldWidget {
	switch info.Kind {
	case proto.FieldKindScalar:
		return NewScalarWidget(info.Desc.Kind())
	case proto.FieldKindBool:
		return NewBoolWidget()
	case proto.FieldKindEnum:
		return NewEnumWidget(info.EnumValues)
	case proto.FieldKindTimestamp:
		return NewTimestampWidget()
	case proto.FieldKindDuration:
		return NewDurationWidget()
	case proto.FieldKindStruct:
		return NewStructWidget()
	case proto.FieldKindWrapper:
		innerKind := protoreflect.StringKind
		if info.Desc != nil && info.Desc.Message() != nil {
			innerKind = wrapperInnerKind(string(info.Desc.Message().FullName()))
		}
		return NewWrapperWidget(innerKind)
	default:
		return NewScalarWidget(protoreflect.StringKind)
	}
}

// AddRepeatedEntry adds a new entry to a repeated field.
func AddRepeatedEntry(field *FormField) {
	if field.Info.Kind != proto.FieldKindRepeated {
		return
	}

	if field.Info.Desc.Kind() == protoreflect.MessageKind {
		// Repeated message: add a nested message form.
		child := FormField{
			Info:     proto.FieldInfo{Name: field.Info.Name, Kind: proto.FieldKindMessage},
			Expanded: true,
			Children: generateFields(field.Info.Children),
		}
		field.Children = append(field.Children, child)
	} else {
		// Repeated scalar: add a scalar input.
		child := FormField{
			Info:   proto.FieldInfo{Name: field.Info.Name, Kind: proto.FieldKindScalar},
			Widget: NewScalarWidget(field.Info.Desc.Kind()),
		}
		field.Children = append(field.Children, child)
	}
}

// AddMapEntry adds a new key-value entry to a map field.
func AddMapEntry(field *FormField) {
	if field.Info.Kind != proto.FieldKindMap {
		return
	}

	var keyWidget, valWidget FieldWidget
	if len(field.Info.Children) >= 2 {
		keyWidget = createWidgetForField(field.Info.Children[0])
		valWidget = createWidgetForField(field.Info.Children[1])
	} else {
		keyWidget = NewScalarWidget(protoreflect.StringKind)
		valWidget = NewScalarWidget(protoreflect.StringKind)
	}

	entry := FormField{
		Info: proto.FieldInfo{Name: field.Info.Name, Kind: proto.FieldKindMap},
		Children: []FormField{
			{Info: proto.FieldInfo{Name: "key"}, Widget: keyWidget},
			{Info: proto.FieldInfo{Name: "value"}, Widget: valWidget},
		},
	}
	field.Children = append(field.Children, entry)
}

// RemoveEntry removes the child at the given index.
func RemoveEntry(field *FormField, idx int) {
	if idx < 0 || idx >= len(field.Children) {
		return
	}
	field.Children = append(field.Children[:idx], field.Children[idx+1:]...)
}
