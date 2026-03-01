package requestbuilder

import (
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"

	"github.com/futuramacoder/protopilot/internal/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const testdataDir = "../../proto/testdata"

func loadMethod(t *testing.T, service, method string) protoreflect.MethodDescriptor {
	t.Helper()
	loader := proto.NewLoader([]string{testdataDir})
	reg, _, err := loader.Load([]string{filepath.Join(testdataDir, "basic.proto")})
	if err != nil {
		t.Fatalf("failed to load proto: %v", err)
	}
	m, ok := reg.GetMethod(protoreflect.FullName(service), protoreflect.Name(method))
	if !ok {
		t.Fatalf("method %s/%s not found", service, method)
	}
	return m.Desc
}

func TestGenerate_SimpleMessage(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "GetUser")
	fields := Generate(md.Input())

	if len(fields) != 1 {
		t.Fatalf("expected 1 field (user_id), got %d", len(fields))
	}
	if fields[0].Info.Name != "user_id" {
		t.Errorf("expected field name 'user_id', got %q", fields[0].Info.Name)
	}
	if fields[0].Widget == nil {
		t.Error("widget should not be nil")
	}
	if _, ok := fields[0].Widget.(*ScalarWidget); !ok {
		t.Errorf("expected ScalarWidget, got %T", fields[0].Widget)
	}
}

func TestGenerate_NestedMessage(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "CreateUser")
	fields := Generate(md.Input())

	// Find the address field.
	var addressField *FormField
	for i := range fields {
		if fields[i].Info.Name == "address" {
			addressField = &fields[i]
			break
		}
	}

	if addressField == nil {
		t.Fatal("address field not found")
	}
	if addressField.Info.Kind != proto.FieldKindMessage {
		t.Errorf("expected FieldKindMessage, got %v", addressField.Info.Kind)
	}
	if len(addressField.Children) == 0 {
		t.Error("address should have children")
	}

	// Verify nested field paths.
	childNames := make(map[string]bool)
	for _, c := range addressField.Children {
		childNames[c.Info.Name] = true
	}
	for _, name := range []string{"street", "city", "state", "zip_code", "country"} {
		if !childNames[name] {
			t.Errorf("nested field %q not found in address", name)
		}
	}
}

func TestGenerate_OneofFields(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "CreateUser")
	fields := Generate(md.Input())

	var oneofField *FormField
	for i := range fields {
		if fields[i].Info.Kind == proto.FieldKindOneof {
			oneofField = &fields[i]
			break
		}
	}

	if oneofField == nil {
		t.Fatal("oneof field not found")
	}
	if oneofField.Info.Name != "contact" {
		t.Errorf("expected oneof name 'contact', got %q", oneofField.Info.Name)
	}
	if len(oneofField.Children) != 2 {
		t.Fatalf("expected 2 oneof variants, got %d", len(oneofField.Children))
	}

	// First variant should be active by default.
	if !oneofField.Children[0].OneofActive {
		t.Error("first oneof variant should be active")
	}
	if oneofField.Children[1].OneofActive {
		t.Error("second oneof variant should not be active")
	}

	// Both should have widgets.
	for _, c := range oneofField.Children {
		if c.Widget == nil {
			t.Errorf("oneof variant %q should have a widget", c.Info.Name)
		}
	}
}

func TestGenerate_RepeatedFields(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "CreateUser")
	fields := Generate(md.Input())

	var tagsField *FormField
	for i := range fields {
		if fields[i].Info.Name == "tags" {
			tagsField = &fields[i]
			break
		}
	}

	if tagsField == nil {
		t.Fatal("tags field not found")
	}
	if tagsField.Info.Kind != proto.FieldKindRepeated {
		t.Errorf("expected FieldKindRepeated, got %v", tagsField.Info.Kind)
	}

	// Initially no entries.
	if len(tagsField.Children) != 0 {
		t.Errorf("expected 0 initial children, got %d", len(tagsField.Children))
	}

	// Add entries.
	AddRepeatedEntry(tagsField)
	AddRepeatedEntry(tagsField)
	if len(tagsField.Children) != 2 {
		t.Errorf("expected 2 children after adding, got %d", len(tagsField.Children))
	}

	// Remove entry.
	RemoveEntry(tagsField, 0)
	if len(tagsField.Children) != 1 {
		t.Errorf("expected 1 child after removing, got %d", len(tagsField.Children))
	}
}

func TestGenerate_MapFields(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "CreateUser")
	fields := Generate(md.Input())

	var labelsField *FormField
	for i := range fields {
		if fields[i].Info.Name == "labels" {
			labelsField = &fields[i]
			break
		}
	}

	if labelsField == nil {
		t.Fatal("labels field not found")
	}
	if labelsField.Info.Kind != proto.FieldKindMap {
		t.Errorf("expected FieldKindMap, got %v", labelsField.Info.Kind)
	}

	AddMapEntry(labelsField)
	if len(labelsField.Children) != 1 {
		t.Fatalf("expected 1 map entry, got %d", len(labelsField.Children))
	}
	if len(labelsField.Children[0].Children) != 2 {
		t.Errorf("map entry should have 2 children (key, value), got %d", len(labelsField.Children[0].Children))
	}
}

func TestGenerate_WellKnownTypes(t *testing.T) {
	md := loadMethod(t, "testdata.UserService", "CreateUser")
	fields := Generate(md.Input())

	expected := map[string]interface{}{
		"created_at": (*TimestampWidget)(nil),
		"timeout":    (*DurationWidget)(nil),
		"metadata":   (*StructWidget)(nil),
		"nickname":   (*WrapperWidget)(nil),
	}

	for _, f := range fields {
		if _, ok := expected[f.Info.Name]; ok {
			if f.Widget == nil {
				t.Errorf("field %q should have a widget", f.Info.Name)
				continue
			}
			switch f.Info.Name {
			case "created_at":
				if _, ok := f.Widget.(*TimestampWidget); !ok {
					t.Errorf("created_at should be TimestampWidget, got %T", f.Widget)
				}
			case "timeout":
				if _, ok := f.Widget.(*DurationWidget); !ok {
					t.Errorf("timeout should be DurationWidget, got %T", f.Widget)
				}
			case "metadata":
				if _, ok := f.Widget.(*StructWidget); !ok {
					t.Errorf("metadata should be StructWidget, got %T", f.Widget)
				}
			case "nickname":
				if _, ok := f.Widget.(*WrapperWidget); !ok {
					t.Errorf("nickname should be WrapperWidget, got %T", f.Widget)
				}
			}
		}
	}
}

func TestValidation_NumericRange(t *testing.T) {
	w := NewScalarWidget(protoreflect.Int32Kind)

	// Valid.
	w.SetValue("42")
	if msg := w.Validate(); msg != "" {
		t.Errorf("42 should be valid int32, got %q", msg)
	}

	// Overflow.
	w.SetValue("99999999999")
	if msg := w.Validate(); msg == "" {
		t.Error("99999999999 should be invalid int32")
	}

	// Not a number.
	w.SetValue("abc")
	if msg := w.Validate(); msg == "" {
		t.Error("abc should be invalid int32")
	}

	// Empty is valid (not set).
	w.SetValue("")
	if msg := w.Validate(); msg != "" {
		t.Errorf("empty should be valid, got %q", msg)
	}

	// uint32.
	uw := NewScalarWidget(protoreflect.Uint32Kind)
	uw.SetValue("-1")
	if msg := uw.Validate(); msg == "" {
		t.Error("-1 should be invalid uint32")
	}
}

func TestValidation_Timestamp(t *testing.T) {
	w := NewTimestampWidget()

	w.SetValue("2024-01-15T10:30:00Z")
	if msg := w.Validate(); msg != "" {
		t.Errorf("valid timestamp should pass, got %q", msg)
	}

	w.SetValue("not-a-timestamp")
	if msg := w.Validate(); msg == "" {
		t.Error("invalid timestamp should fail")
	}

	w.SetValue("")
	if msg := w.Validate(); msg != "" {
		t.Errorf("empty should be valid, got %q", msg)
	}
}

func TestValidation_Duration(t *testing.T) {
	w := NewDurationWidget()

	w.SetValue("5s")
	if msg := w.Validate(); msg != "" {
		t.Errorf("5s should be valid, got %q", msg)
	}

	w.SetValue("1m30s")
	if msg := w.Validate(); msg != "" {
		t.Errorf("1m30s should be valid, got %q", msg)
	}

	w.SetValue("not-a-duration")
	if msg := w.Validate(); msg == "" {
		t.Error("invalid duration should fail")
	}
}

func TestValidation_JSON(t *testing.T) {
	w := NewStructWidget()

	w.SetValue(`{"key": "value"}`)
	if msg := w.Validate(); msg != "" {
		t.Errorf("valid JSON should pass, got %q", msg)
	}

	w.SetValue(`{invalid}`)
	if msg := w.Validate(); msg == "" {
		t.Error("invalid JSON should fail")
	}

	w.SetValue("")
	if msg := w.Validate(); msg != "" {
		t.Errorf("empty should be valid, got %q", msg)
	}
}

func TestMetadataSection(t *testing.T) {
	m := NewMetadataSection()

	// Initially empty.
	if len(m.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(m.Entries))
	}
	result := m.ToMap()
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}

	// Add entries.
	m.AddEntry()
	m.AddEntry()
	if len(m.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(m.Entries))
	}

	// Set values.
	m.Entries[0].Key.SetValue("authorization")
	m.Entries[0].Value.SetValue("Bearer token123")
	m.Entries[1].Key.SetValue("x-request-id")
	m.Entries[1].Value.SetValue("abc")

	result = m.ToMap()
	if result["authorization"] != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got %q", result["authorization"])
	}
	if result["x-request-id"] != "abc" {
		t.Errorf("expected 'abc', got %q", result["x-request-id"])
	}

	// Remove entry.
	m.RemoveEntry(0)
	if len(m.Entries) != 1 {
		t.Errorf("expected 1 entry after remove, got %d", len(m.Entries))
	}
	result = m.ToMap()
	if _, ok := result["authorization"]; ok {
		t.Error("authorization should be removed")
	}
}

func TestBuildGrpcurlCommand(t *testing.T) {
	fields := []FormField{
		{
			Info:   proto.FieldInfo{Name: "name"},
			Widget: &ScalarWidget{input: setInputValue("world"), kind: protoreflect.StringKind},
		},
	}

	metadata := map[string]string{
		"authorization": "Bearer token123",
	}

	cmd := BuildGrpcurlCommand(
		"localhost:50051",
		true,
		"helloworld.Greeter",
		"SayHello",
		fields,
		metadata,
		TLSConfig{},
	)

	if !strings.Contains(cmd, "grpcurl") {
		t.Error("should contain grpcurl")
	}
	if !strings.Contains(cmd, "-plaintext") {
		t.Error("should contain -plaintext flag")
	}
	if !strings.Contains(cmd, "helloworld.Greeter/SayHello") {
		t.Error("should contain service/method")
	}
	if !strings.Contains(cmd, "localhost:50051") {
		t.Error("should contain host")
	}
	if !strings.Contains(cmd, "authorization: Bearer token123") {
		t.Error("should contain metadata header")
	}
	if !strings.Contains(cmd, `"name"`) {
		t.Error("should contain field data")
	}

	// Test with TLS config.
	cmd = BuildGrpcurlCommand(
		"secure.example.com:443",
		false,
		"pkg.Service",
		"Method",
		nil,
		nil,
		TLSConfig{
			CACert:     "ca.pem",
			Cert:       "client.pem",
			Key:        "client-key.pem",
			ServerName: "api.example.com",
		},
	)

	if !strings.Contains(cmd, "-cacert ca.pem") {
		t.Error("should contain -cacert flag")
	}
	if !strings.Contains(cmd, "-cert client.pem") {
		t.Error("should contain -cert flag")
	}
	if !strings.Contains(cmd, "-key client-key.pem") {
		t.Error("should contain -key flag")
	}
	if !strings.Contains(cmd, "-servername api.example.com") {
		t.Error("should contain -servername flag")
	}
}

// setInputValue creates a textinput with a pre-set value.
func setInputValue(s string) textinput.Model {
	ti := newTextInput("")
	ti.SetValue(s)
	return ti
}
