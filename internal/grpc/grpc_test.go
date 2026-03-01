package grpc

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const testdataDir = "../../proto/testdata"

func getMessageDescriptor(t *testing.T, msgName string) protoreflect.MessageDescriptor {
	t.Helper()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{testdataDir},
		}),
		Reporter: reporter.NewReporter(nil, nil),
	}
	files, err := compiler.Compile(context.Background(), "basic.proto")
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}
	f := files.FindFileByPath("basic.proto")
	if f == nil {
		t.Fatal("basic.proto not found")
	}
	md := f.Messages().ByName(protoreflect.Name(msgName))
	if md == nil {
		t.Fatalf("message %s not found", msgName)
	}
	return md
}

func getOrdersMessageDescriptor(t *testing.T, msgName string) protoreflect.MessageDescriptor {
	t.Helper()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{testdataDir},
		}),
		Reporter: reporter.NewReporter(nil, nil),
	}
	files, err := compiler.Compile(context.Background(), "orders.proto")
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}
	f := files.FindFileByPath("orders.proto")
	if f == nil {
		t.Fatal("orders.proto not found")
	}
	md := f.Messages().ByName(protoreflect.Name(msgName))
	if md == nil {
		t.Fatalf("message %s not found", msgName)
	}
	return md
}

func TestBuildMessage_Scalars(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	msg, err := BuildMessage(md, map[string]any{
		"name":       "Alice",
		"email":      "alice@example.com",
		"age":        "30",
		"score":      "9999999999",
		"level":      "5",
		"experience": "100000",
		"rating":     "4.5",
		"balance":    "1234.56",
		"active":     "true",
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	// Verify by marshaling to JSON and checking.
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	checks := []string{`"name":"Alice"`, `"email":"alice@example.com"`, `"active":true`}
	for _, check := range checks {
		if !containsStr(jsonStr, check) {
			t.Errorf("JSON should contain %s, got: %s", check, jsonStr)
		}
	}
}

func TestBuildMessage_Nested(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	msg, err := BuildMessage(md, map[string]any{
		"name": "Bob",
		"address": map[string]any{
			"street":  "123 Main St",
			"city":    "Springfield",
			"state":   "IL",
			"zip_code": "62701",
			"country": "US",
		},
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	if !containsStr(jsonStr, `"street":"123 Main St"`) {
		t.Errorf("JSON should contain nested street, got: %s", jsonStr)
	}
	if !containsStr(jsonStr, `"city":"Springfield"`) {
		t.Errorf("JSON should contain nested city, got: %s", jsonStr)
	}
}

func TestBuildMessage_Repeated(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	msg, err := BuildMessage(md, map[string]any{
		"name": "Charlie",
		"tags": []any{"admin", "active", "premium"},
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	if !containsStr(jsonStr, `"tags"`) {
		t.Errorf("JSON should contain tags, got: %s", jsonStr)
	}
	if !containsStr(jsonStr, `"admin"`) {
		t.Errorf("JSON should contain admin tag, got: %s", jsonStr)
	}
}

func TestBuildMessage_Map(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	msg, err := BuildMessage(md, map[string]any{
		"name": "Diana",
		"labels": map[string]any{
			"env":  "prod",
			"team": "backend",
		},
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	if !containsStr(jsonStr, `"labels"`) {
		t.Errorf("JSON should contain labels, got: %s", jsonStr)
	}
}

func TestBuildMessage_Enum(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	msg, err := BuildMessage(md, map[string]any{
		"name": "Eve",
		"role": "USER_ROLE_ADMIN",
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	jsonStr := string(jsonBytes)

	if !containsStr(jsonStr, `"role":"USER_ROLE_ADMIN"`) {
		t.Errorf("JSON should contain enum value, got: %s", jsonStr)
	}
}

func TestBuildMessage_WellKnownTypes(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")

	t.Run("Timestamp", func(t *testing.T) {
		msg, err := BuildMessage(md, map[string]any{
			"name":       "Frank",
			"created_at": "2024-01-15T10:30:00Z",
		})
		if err != nil {
			t.Fatalf("BuildMessage failed: %v", err)
		}
		jsonBytes, _ := protojson.Marshal(msg)
		jsonStr := string(jsonBytes)
		if !containsStr(jsonStr, `"createdAt"`) {
			t.Errorf("JSON should contain createdAt, got: %s", jsonStr)
		}
	})

	t.Run("Duration", func(t *testing.T) {
		msg, err := BuildMessage(md, map[string]any{
			"name":    "Grace",
			"timeout": "30s",
		})
		if err != nil {
			t.Fatalf("BuildMessage failed: %v", err)
		}
		jsonBytes, _ := protojson.Marshal(msg)
		jsonStr := string(jsonBytes)
		if !containsStr(jsonStr, `"timeout"`) {
			t.Errorf("JSON should contain timeout, got: %s", jsonStr)
		}
	})

	t.Run("Struct", func(t *testing.T) {
		msg, err := BuildMessage(md, map[string]any{
			"name":     "Hank",
			"metadata": `{"key": "value", "count": 42}`,
		})
		if err != nil {
			t.Fatalf("BuildMessage failed: %v", err)
		}
		jsonBytes, _ := protojson.Marshal(msg)
		jsonStr := string(jsonBytes)
		if !containsStr(jsonStr, `"metadata"`) {
			t.Errorf("JSON should contain metadata, got: %s", jsonStr)
		}
	})

	t.Run("Wrapper", func(t *testing.T) {
		msg, err := BuildMessage(md, map[string]any{
			"name":     "Ivy",
			"nickname": "Ivy-girl",
		})
		if err != nil {
			t.Fatalf("BuildMessage failed: %v", err)
		}
		jsonBytes, _ := protojson.Marshal(msg)
		jsonStr := string(jsonBytes)
		if !containsStr(jsonStr, `"nickname"`) {
			t.Errorf("JSON should contain nickname, got: %s", jsonStr)
		}
	})
}

func TestBuildMessage_RepeatedMessage(t *testing.T) {
	md := getOrdersMessageDescriptor(t, "CreateOrderRequest")

	msg, err := BuildMessage(md, map[string]any{
		"user_id": "user-123",
		"items": []any{
			map[string]any{
				"product_id": "prod-1",
				"name":       "Widget",
				"quantity":   "2",
				"price":      "9.99",
			},
		},
		"shipping_address": "123 Main St",
	})
	if err != nil {
		t.Fatalf("BuildMessage failed: %v", err)
	}

	jsonBytes, _ := protojson.Marshal(msg)
	jsonStr := string(jsonBytes)
	if !containsStr(jsonStr, `"productId":"prod-1"`) {
		t.Errorf("JSON should contain product_id, got: %s", jsonStr)
	}
}

func TestCoerceValue_AllTypes(t *testing.T) {
	md := getMessageDescriptor(t, "CreateUserRequest")
	fields := md.Fields()

	tests := []struct {
		fieldName string
		input     string
		wantErr   bool
	}{
		{"name", "test", false},
		{"age", "42", false},
		{"age", "abc", true},
		{"score", "999", false},
		{"level", "5", false},
		{"level", "-1", true}, // uint32 can't be negative
		{"experience", "100", false},
		{"rating", "3.14", false},
		{"balance", "1.23", false},
		{"active", "true", false},
		{"active", "yes", true},
		{"role", "USER_ROLE_ADMIN", false},
		{"role", "NONEXISTENT", true},
	}

	for _, tt := range tests {
		fd := fields.ByName(protoreflect.Name(tt.fieldName))
		if fd == nil {
			t.Errorf("field %s not found", tt.fieldName)
			continue
		}
		_, err := coerceValue(fd, tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("coerceValue(%s, %q) should error", tt.fieldName, tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("coerceValue(%s, %q) unexpected error: %v", tt.fieldName, tt.input, err)
		}
	}
}

func TestDecodeErrorDetails_Empty(t *testing.T) {
	details := DecodeErrorDetails(nil)
	if len(details) != 0 {
		t.Errorf("expected no details for nil status, got %d", len(details))
	}
}

func TestClient_NewClient(t *testing.T) {
	c := NewClient("localhost:50051", TLSConfig{Plaintext: true})
	if c.Host() != "localhost:50051" {
		t.Errorf("expected host localhost:50051, got %s", c.Host())
	}
	if !c.TLS().Plaintext {
		t.Error("expected plaintext to be true")
	}
	if c.Conn() != nil {
		t.Error("connection should be nil before connect")
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && filepath.Base(s) != "" && // avoid unused import
		stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
