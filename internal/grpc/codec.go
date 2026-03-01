package grpc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BuildMessage converts form field values into a dynamicpb.Message.
// The fieldValues map uses field names as keys (not dot-notation).
func BuildMessage(
	md protoreflect.MessageDescriptor,
	fieldValues map[string]any,
) (*dynamicpb.Message, error) {
	msg := dynamicpb.NewMessage(md)

	for name, val := range fieldValues {
		fd := md.Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			continue
		}

		if err := setField(msg, fd, val); err != nil {
			return nil, fmt.Errorf("field %s: %w", name, err)
		}
	}

	return msg, nil
}

func setField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	// Handle map fields.
	if fd.IsMap() {
		mapVal, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map for field %s", fd.Name())
		}
		mapField := msg.Mutable(fd).Map()
		keyDesc := fd.MapKey()
		valDesc := fd.MapValue()
		for k, v := range mapVal {
			keyVal, err := coerceValue(keyDesc, k)
			if err != nil {
				return fmt.Errorf("map key: %w", err)
			}
			valVal, err := coerceValue(valDesc, v)
			if err != nil {
				return fmt.Errorf("map value: %w", err)
			}
			mapField.Set(keyVal.MapKey(), valVal)
		}
		return nil
	}

	// Handle repeated fields.
	if fd.IsList() {
		listVal, ok := val.([]any)
		if !ok {
			return fmt.Errorf("expected slice for repeated field %s", fd.Name())
		}
		list := msg.Mutable(fd).List()
		for _, item := range listVal {
			if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
				nested, ok := item.(map[string]any)
				if !ok {
					return fmt.Errorf("expected map for repeated message item in %s", fd.Name())
				}
				innerMsg, err := BuildMessage(fd.Message(), nested)
				if err != nil {
					return fmt.Errorf("repeated message item: %w", err)
				}
				list.Append(protoreflect.ValueOfMessage(innerMsg))
			} else {
				v, err := coerceValue(fd, item)
				if err != nil {
					return fmt.Errorf("repeated item: %w", err)
				}
				list.Append(v)
			}
		}
		return nil
	}

	// Handle nested messages.
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		return setMessageField(msg, fd, val)
	}

	// Scalar field.
	v, err := coerceValue(fd, val)
	if err != nil {
		return err
	}
	msg.Set(fd, v)
	return nil
}

func setMessageField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	fullName := fd.Message().FullName()

	switch fullName {
	case "google.protobuf.Timestamp":
		return setTimestampField(msg, fd, val)
	case "google.protobuf.Duration":
		return setDurationField(msg, fd, val)
	case "google.protobuf.Struct":
		return setStructField(msg, fd, val)
	case "google.protobuf.Value":
		return setValueField(msg, fd, val)
	}

	// Check wrapper types.
	if isWrapperType(fullName) {
		return setWrapperField(msg, fd, val)
	}

	// Regular nested message.
	nested, ok := val.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map for message field %s", fd.Name())
	}
	innerMsg, err := BuildMessage(fd.Message(), nested)
	if err != nil {
		return err
	}
	msg.Set(fd, protoreflect.ValueOfMessage(innerMsg))
	return nil
}

func setTimestampField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string for timestamp")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	ts := timestamppb.New(t)
	inner := dynamicpb.NewMessage(fd.Message())
	inner.Set(fd.Message().Fields().ByName("seconds"), protoreflect.ValueOfInt64(ts.Seconds))
	inner.Set(fd.Message().Fields().ByName("nanos"), protoreflect.ValueOfInt32(ts.Nanos))
	msg.Set(fd, protoreflect.ValueOfMessage(inner))
	return nil
}

func setDurationField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string for duration")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	dp := durationpb.New(d)
	inner := dynamicpb.NewMessage(fd.Message())
	inner.Set(fd.Message().Fields().ByName("seconds"), protoreflect.ValueOfInt64(dp.Seconds))
	inner.Set(fd.Message().Fields().ByName("nanos"), protoreflect.ValueOfInt32(dp.Nanos))
	msg.Set(fd, protoreflect.ValueOfMessage(inner))
	return nil
}

func setStructField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected JSON string for struct")
	}
	st := &structpb.Struct{}
	if err := json.Unmarshal([]byte(s), st); err != nil {
		return fmt.Errorf("invalid JSON for struct: %w", err)
	}
	// Use protojson round-trip via dynamicpb.
	inner := dynamicpb.NewMessage(fd.Message())
	buildStructMessage(inner, fd.Message(), st)
	msg.Set(fd, protoreflect.ValueOfMessage(inner))
	return nil
}

func setValueField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected JSON string for value")
	}
	v := &structpb.Value{}
	if err := json.Unmarshal([]byte(s), v); err != nil {
		return fmt.Errorf("invalid JSON for value: %w", err)
	}
	inner := dynamicpb.NewMessage(fd.Message())
	buildValueMessage(inner, fd.Message(), v)
	msg.Set(fd, protoreflect.ValueOfMessage(inner))
	return nil
}

func setWrapperField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, val any) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string for wrapper")
	}
	if s == "" {
		return nil // null / not set
	}
	inner := dynamicpb.NewMessage(fd.Message())
	valueField := fd.Message().Fields().ByName("value")
	if valueField == nil {
		return fmt.Errorf("wrapper type missing 'value' field")
	}
	v, err := coerceValue(valueField, s)
	if err != nil {
		return fmt.Errorf("wrapper value: %w", err)
	}
	inner.Set(valueField, v)
	msg.Set(fd, protoreflect.ValueOfMessage(inner))
	return nil
}

// coerceValue converts a string input value to the appropriate protoreflect.Value.
func coerceValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	s := fmt.Sprintf("%v", val)

	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(s), nil

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid int32: %w", err)
		}
		return protoreflect.ValueOfInt32(int32(n)), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid int64: %w", err)
		}
		return protoreflect.ValueOfInt64(n), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid uint32: %w", err)
		}
		return protoreflect.ValueOfUint32(uint32(n)), nil

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid uint64: %w", err)
		}
		return protoreflect.ValueOfUint64(n), nil

	case protoreflect.FloatKind:
		f, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid float: %w", err)
		}
		return protoreflect.ValueOfFloat32(float32(f)), nil

	case protoreflect.DoubleKind:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid double: %w", err)
		}
		return protoreflect.ValueOfFloat64(f), nil

	case protoreflect.BoolKind:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid bool: %w", err)
		}
		return protoreflect.ValueOfBool(b), nil

	case protoreflect.BytesKind:
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid base64: %w", err)
		}
		return protoreflect.ValueOfBytes(b), nil

	case protoreflect.EnumKind:
		enumDesc := fd.Enum()
		v := enumDesc.Values().ByName(protoreflect.Name(s))
		if v == nil {
			// Try parsing as number.
			n, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return protoreflect.Value{}, fmt.Errorf("unknown enum value: %s", s)
			}
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil
		}
		return protoreflect.ValueOfEnum(v.Number()), nil

	case protoreflect.MessageKind, protoreflect.GroupKind:
		// Should be handled at a higher level.
		return protoreflect.Value{}, fmt.Errorf("message fields must be handled separately")

	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind: %v", fd.Kind())
	}
}

func isWrapperType(fullName protoreflect.FullName) bool {
	switch fullName {
	case "google.protobuf.DoubleValue", "google.protobuf.FloatValue",
		"google.protobuf.Int64Value", "google.protobuf.UInt64Value",
		"google.protobuf.Int32Value", "google.protobuf.UInt32Value",
		"google.protobuf.BoolValue", "google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return true
	}
	return false
}

// buildStructMessage populates a dynamic Struct message from a structpb.Struct.
func buildStructMessage(msg *dynamicpb.Message, md protoreflect.MessageDescriptor, st *structpb.Struct) {
	fieldsDesc := md.Fields().ByName("fields")
	if fieldsDesc == nil || st == nil {
		return
	}
	mapField := msg.Mutable(fieldsDesc).Map()
	for k, v := range st.Fields {
		valMsg := dynamicpb.NewMessage(fieldsDesc.MapValue().Message())
		buildValueMessage(valMsg, fieldsDesc.MapValue().Message(), v)
		mapField.Set(protoreflect.ValueOfString(k).MapKey(), protoreflect.ValueOfMessage(valMsg))
	}
}

// buildValueMessage populates a dynamic Value message from a structpb.Value.
func buildValueMessage(msg *dynamicpb.Message, md protoreflect.MessageDescriptor, v *structpb.Value) {
	if v == nil {
		return
	}

	switch k := v.Kind.(type) {
	case *structpb.Value_NullValue:
		nullField := md.Fields().ByName("null_value")
		if nullField != nil {
			msg.Set(nullField, protoreflect.ValueOfEnum(0))
		}
	case *structpb.Value_NumberValue:
		numField := md.Fields().ByName("number_value")
		if numField != nil {
			msg.Set(numField, protoreflect.ValueOfFloat64(k.NumberValue))
		}
	case *structpb.Value_StringValue:
		strField := md.Fields().ByName("string_value")
		if strField != nil {
			msg.Set(strField, protoreflect.ValueOfString(k.StringValue))
		}
	case *structpb.Value_BoolValue:
		boolField := md.Fields().ByName("bool_value")
		if boolField != nil {
			msg.Set(boolField, protoreflect.ValueOfBool(k.BoolValue))
		}
	case *structpb.Value_StructValue:
		structField := md.Fields().ByName("struct_value")
		if structField != nil {
			innerMsg := dynamicpb.NewMessage(structField.Message())
			buildStructMessage(innerMsg, structField.Message(), k.StructValue)
			msg.Set(structField, protoreflect.ValueOfMessage(innerMsg))
		}
	case *structpb.Value_ListValue:
		listField := md.Fields().ByName("list_value")
		if listField != nil {
			innerMsg := dynamicpb.NewMessage(listField.Message())
			valuesDesc := listField.Message().Fields().ByName("values")
			if valuesDesc != nil {
				list := innerMsg.Mutable(valuesDesc).List()
				for _, item := range k.ListValue.Values {
					itemMsg := dynamicpb.NewMessage(valuesDesc.Message())
					buildValueMessage(itemMsg, valuesDesc.Message(), item)
					list.Append(protoreflect.ValueOfMessage(itemMsg))
				}
			}
			msg.Set(listField, protoreflect.ValueOfMessage(innerMsg))
		}
	}
}

// Ensure math is used (for potential future float validation).
var _ = math.MaxFloat32
