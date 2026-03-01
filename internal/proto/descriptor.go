package proto

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FieldKind classifies proto fields for the request builder.
type FieldKind int

const (
	FieldKindScalar    FieldKind = iota // string, int32, int64, float, double, bytes
	FieldKindBool                       // bool → toggle widget
	FieldKindEnum                       // enum → popup list widget
	FieldKindMessage                    // nested message → collapsible section
	FieldKindRepeated                   // repeated field → collapsible list
	FieldKindMap                        // map field → key-value rows
	FieldKindOneof                      // oneof group → radio selector
	FieldKindTimestamp                  // google.protobuf.Timestamp → datetime input
	FieldKindDuration                   // google.protobuf.Duration → duration input
	FieldKindStruct                     // google.protobuf.Struct/Value → JSON editor
	FieldKindWrapper                    // google.protobuf.StringValue etc. → nullable input
)

// String returns a human-readable name for the FieldKind.
func (k FieldKind) String() string {
	switch k {
	case FieldKindScalar:
		return "Scalar"
	case FieldKindBool:
		return "Bool"
	case FieldKindEnum:
		return "Enum"
	case FieldKindMessage:
		return "Message"
	case FieldKindRepeated:
		return "Repeated"
	case FieldKindMap:
		return "Map"
	case FieldKindOneof:
		return "Oneof"
	case FieldKindTimestamp:
		return "Timestamp"
	case FieldKindDuration:
		return "Duration"
	case FieldKindStruct:
		return "Struct"
	case FieldKindWrapper:
		return "Wrapper"
	default:
		return "Unknown"
	}
}

// Well-known type full names.
const (
	timestampFullName   = "google.protobuf.Timestamp"
	durationFullName    = "google.protobuf.Duration"
	structFullName      = "google.protobuf.Struct"
	valueFullName       = "google.protobuf.Value"
	fieldMaskFullName   = "google.protobuf.FieldMask"
	listValueFullName   = "google.protobuf.ListValue"
)

// Wrapper type full names.
var wrapperFullNames = map[protoreflect.FullName]bool{
	"google.protobuf.DoubleValue": true,
	"google.protobuf.FloatValue":  true,
	"google.protobuf.Int64Value":  true,
	"google.protobuf.UInt64Value": true,
	"google.protobuf.Int32Value":  true,
	"google.protobuf.UInt32Value": true,
	"google.protobuf.BoolValue":   true,
	"google.protobuf.StringValue": true,
	"google.protobuf.BytesValue":  true,
}

// FieldInfo describes a single field for form generation.
type FieldInfo struct {
	Path       string                             // Dot-notation path (e.g., "address.street")
	Name       string                             // Field name
	Kind       FieldKind                          // Classification
	Desc       protoreflect.FieldDescriptor       // Original descriptor
	EnumValues []protoreflect.EnumValueDescriptor // Non-nil for FieldKindEnum
	Children   []FieldInfo                        // Non-nil for FieldKindMessage, FieldKindMap
	OneofPeers []FieldInfo                        // Non-nil for FieldKindOneof
}

// ClassifyField determines the FieldKind for a given field descriptor.
//
// Classification priority (highest first):
//  1. Well-known types (Timestamp, Duration, Struct, Value, wrappers)
//  2. Map
//  3. Repeated
//  4. Oneof (for the group; inner fields classified normally)
//  5. Message
//  6. Enum
//  7. Bool
//  8. Everything else → Scalar
func ClassifyField(fd protoreflect.FieldDescriptor) FieldKind {
	// 1. Check for well-known message types.
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		if !fd.IsMap() && !fd.IsList() {
			if kind, ok := classifyWellKnown(fd.Message()); ok {
				return kind
			}
		}
	}

	// 2. Map (must check before repeated since maps are also repeated).
	if fd.IsMap() {
		return FieldKindMap
	}

	// 3. Repeated.
	if fd.IsList() {
		return FieldKindRepeated
	}

	// 5. Message.
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		return FieldKindMessage
	}

	// 6. Enum.
	if fd.Kind() == protoreflect.EnumKind {
		return FieldKindEnum
	}

	// 7. Bool.
	if fd.Kind() == protoreflect.BoolKind {
		return FieldKindBool
	}

	// 8. Everything else.
	return FieldKindScalar
}

// classifyWellKnown returns the FieldKind for a well-known message type.
func classifyWellKnown(md protoreflect.MessageDescriptor) (FieldKind, bool) {
	fullName := md.FullName()

	switch fullName {
	case timestampFullName:
		return FieldKindTimestamp, true
	case durationFullName:
		return FieldKindDuration, true
	case structFullName, valueFullName, listValueFullName:
		return FieldKindStruct, true
	}

	if wrapperFullNames[fullName] {
		return FieldKindWrapper, true
	}

	return 0, false
}

// IsWellKnownType returns true if the message is a recognized well-known type.
func IsWellKnownType(md protoreflect.MessageDescriptor) bool {
	_, ok := classifyWellKnown(md)
	return ok
}

// WalkMessage recursively walks a message descriptor and returns
// a list of FieldInfo entries for form generation. Oneof groups are
// represented as a single FieldKindOneof entry with OneofPeers containing
// the individual variants.
func WalkMessage(md protoreflect.MessageDescriptor) []FieldInfo {
	return walkFields(md, "")
}

func walkFields(md protoreflect.MessageDescriptor, prefix string) []FieldInfo {
	fields := md.Fields()
	oneofs := md.Oneofs()

	// Track which fields belong to a oneof so we don't duplicate them.
	oneofFieldSeen := make(map[protoreflect.FullName]bool)

	var result []FieldInfo

	// Process oneof groups first, collecting their member fields.
	for i := 0; i < oneofs.Len(); i++ {
		oo := oneofs.Get(i)

		// Skip synthetic oneofs (proto3 optional).
		if oo.IsSynthetic() {
			continue
		}

		ooFields := oo.Fields()
		var peers []FieldInfo
		for j := 0; j < ooFields.Len(); j++ {
			fd := ooFields.Get(j)
			oneofFieldSeen[fd.FullName()] = true
			peers = append(peers, classifyFieldInfo(fd, prefix))
		}

		path := prefix + string(oo.Name())
		result = append(result, FieldInfo{
			Path:       path,
			Name:       string(oo.Name()),
			Kind:       FieldKindOneof,
			OneofPeers: peers,
		})
	}

	// Process remaining fields (non-oneof).
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		if oneofFieldSeen[fd.FullName()] {
			continue
		}

		// Skip synthetic oneof fields (proto3 optional) — treat them as normal fields.
		if fd.ContainingOneof() != nil && fd.ContainingOneof().IsSynthetic() {
			// Fall through to normal classification.
		}

		result = append(result, classifyFieldInfo(fd, prefix))
	}

	return result
}

// classifyFieldInfo creates a FieldInfo for a single field descriptor.
func classifyFieldInfo(fd protoreflect.FieldDescriptor, prefix string) FieldInfo {
	path := prefix + string(fd.Name())
	kind := ClassifyField(fd)

	info := FieldInfo{
		Path: path,
		Name: string(fd.Name()),
		Kind: kind,
		Desc: fd,
	}

	switch kind {
	case FieldKindEnum:
		enumDesc := fd.Enum()
		values := enumDesc.Values()
		info.EnumValues = make([]protoreflect.EnumValueDescriptor, values.Len())
		for i := 0; i < values.Len(); i++ {
			info.EnumValues[i] = values.Get(i)
		}

	case FieldKindMessage:
		info.Children = walkFields(fd.Message(), path+".")

	case FieldKindMap:
		mapEntry := fd.Message()
		keyField := mapEntry.Fields().ByName("key")
		valField := mapEntry.Fields().ByName("value")
		if keyField != nil && valField != nil {
			info.Children = []FieldInfo{
				classifyFieldInfo(keyField, path+"."),
				classifyFieldInfo(valField, path+"."),
			}
		}

	case FieldKindRepeated:
		// For repeated messages, include children for the element type.
		if fd.Kind() == protoreflect.MessageKind {
			info.Children = walkFields(fd.Message(), path+"[].")
		}
	}

	return info
}
