package proto

import (
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

const testdataDir = "../../proto/testdata"

func newTestLoader() *Loader {
	return NewLoader([]string{testdataDir})
}

func TestLoad_BasicProto(t *testing.T) {
	loader := newTestLoader()
	reg, warnings, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	pkgs := reg.Packages()
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	pkg := pkgs[0]
	if pkg.Name != "testdata" {
		t.Errorf("expected package 'testdata', got %q", pkg.Name)
	}
	if len(pkg.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(pkg.Services))
	}

	svc := pkg.Services[0]
	if svc.Name != "testdata.UserService" {
		t.Errorf("expected service 'testdata.UserService', got %q", svc.Name)
	}
	if len(svc.Methods) != 3 {
		t.Fatalf("expected 3 methods, got %d", len(svc.Methods))
	}

	// Verify method names.
	methodNames := make(map[protoreflect.Name]bool)
	for _, m := range svc.Methods {
		methodNames[m.Name] = true
	}
	for _, name := range []protoreflect.Name{"GetUser", "CreateUser", "ListUsers"} {
		if !methodNames[name] {
			t.Errorf("method %q not found", name)
		}
	}

	// Verify streaming detection.
	for _, m := range svc.Methods {
		if m.Name == "ListUsers" && !m.IsStreaming {
			t.Error("ListUsers should be streaming")
		}
		if m.Name == "GetUser" && m.IsStreaming {
			t.Error("GetUser should not be streaming")
		}
		if m.Name == "CreateUser" && m.IsStreaming {
			t.Error("CreateUser should not be streaming")
		}
	}
}

func TestLoad_PackageMerge(t *testing.T) {
	loader := newTestLoader()
	reg, warnings, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
		filepath.Join(testdataDir, "orders.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	pkgs := reg.Packages()
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package (merged), got %d: %v", len(pkgs), pkgNames(pkgs))
	}

	pkg := pkgs[0]
	if pkg.Name != "testdata" {
		t.Errorf("expected package 'testdata', got %q", pkg.Name)
	}
	if len(pkg.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(pkg.Services))
	}

	svcNames := make(map[protoreflect.FullName]bool)
	for _, svc := range pkg.Services {
		svcNames[svc.Name] = true
	}
	if !svcNames["testdata.UserService"] {
		t.Error("UserService not found")
	}
	if !svcNames["testdata.OrderService"] {
		t.Error("OrderService not found")
	}
}

func TestLoad_MultiPackage(t *testing.T) {
	loader := newTestLoader()
	reg, warnings, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
		filepath.Join(testdataDir, "orders.proto"),
		filepath.Join(testdataDir, "separate_pkg.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	pkgs := reg.Packages()
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), pkgNames(pkgs))
	}

	// Packages should be sorted alphabetically.
	if pkgs[0].Name != "payments" {
		t.Errorf("expected first package 'payments', got %q", pkgs[0].Name)
	}
	if pkgs[1].Name != "testdata" {
		t.Errorf("expected second package 'testdata', got %q", pkgs[1].Name)
	}

	// testdata package should have 2 services (merged).
	for _, pkg := range pkgs {
		if pkg.Name == "testdata" && len(pkg.Services) != 2 {
			t.Errorf("testdata should have 2 services, got %d", len(pkg.Services))
		}
		if pkg.Name == "payments" && len(pkg.Services) != 1 {
			t.Errorf("payments should have 1 service, got %d", len(pkg.Services))
		}
	}
}

func TestLoad_PartialFailure(t *testing.T) {
	loader := newTestLoader()
	reg, warnings, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
		filepath.Join(testdataDir, "nonexistent.proto"),
	})
	if err != nil {
		t.Fatalf("Load should not fail entirely: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for failed file")
	}
	if reg == nil {
		t.Fatal("registry should not be nil on partial failure")
	}

	pkgs := reg.Packages()
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package from partial load, got %d", len(pkgs))
	}
}

func TestLoad_AllFail(t *testing.T) {
	loader := newTestLoader()
	_, _, err := loader.Load([]string{
		filepath.Join(testdataDir, "nonexistent1.proto"),
		filepath.Join(testdataDir, "nonexistent2.proto"),
	})
	if err == nil {
		t.Error("expected error when all files fail")
	}
}

func TestGetMethod(t *testing.T) {
	loader := newTestLoader()
	reg, _, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Existing method.
	m, ok := reg.GetMethod("testdata.UserService", "GetUser")
	if !ok {
		t.Fatal("GetUser not found")
	}
	if m.Name != "GetUser" {
		t.Errorf("expected 'GetUser', got %q", m.Name)
	}
	if m.IsStreaming {
		t.Error("GetUser should not be streaming")
	}

	// Non-existent method.
	_, ok = reg.GetMethod("testdata.UserService", "NonExistent")
	if ok {
		t.Error("should not find non-existent method")
	}

	// Non-existent service.
	_, ok = reg.GetMethod("testdata.NonExistent", "GetUser")
	if ok {
		t.Error("should not find method on non-existent service")
	}
}

func TestClassifyField(t *testing.T) {
	loader := newTestLoader()
	reg, _, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	m, ok := reg.GetMethod("testdata.UserService", "CreateUser")
	if !ok {
		t.Fatal("CreateUser not found")
	}

	inputDesc := m.Desc.Input()
	fields := inputDesc.Fields()

	tests := []struct {
		name string
		want FieldKind
	}{
		{"name", FieldKindScalar},
		{"email", FieldKindScalar},
		{"role", FieldKindEnum},
		{"address", FieldKindMessage},
		{"tags", FieldKindRepeated},
		{"labels", FieldKindMap},
		{"created_at", FieldKindTimestamp},
		{"timeout", FieldKindDuration},
		{"metadata", FieldKindStruct},
		{"nickname", FieldKindWrapper},
		{"age", FieldKindScalar},
		{"score", FieldKindScalar},
		{"level", FieldKindScalar},
		{"experience", FieldKindScalar},
		{"rating", FieldKindScalar},
		{"balance", FieldKindScalar},
		{"active", FieldKindBool},
		{"avatar", FieldKindScalar},
		// oneof members are classified by their inner type
		{"phone", FieldKindScalar},
		{"slack_handle", FieldKindScalar},
	}

	for _, tt := range tests {
		fd := fields.ByName(protoreflect.Name(tt.name))
		if fd == nil {
			t.Errorf("field %q not found", tt.name)
			continue
		}
		got := ClassifyField(fd)
		if got != tt.want {
			t.Errorf("ClassifyField(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestWalkMessage(t *testing.T) {
	loader := newTestLoader()
	reg, _, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	m, ok := reg.GetMethod("testdata.UserService", "CreateUser")
	if !ok {
		t.Fatal("CreateUser not found")
	}

	infos := WalkMessage(m.Desc.Input())
	if len(infos) == 0 {
		t.Fatal("WalkMessage returned empty")
	}

	// Build a map of path → kind for easy lookup.
	byPath := make(map[string]FieldKind)
	collectPaths(infos, byPath)

	// Verify top-level fields.
	expectedTopLevel := map[string]FieldKind{
		"name":        FieldKindScalar,
		"email":       FieldKindScalar,
		"role":        FieldKindEnum,
		"address":     FieldKindMessage,
		"tags":        FieldKindRepeated,
		"labels":      FieldKindMap,
		"created_at":  FieldKindTimestamp,
		"timeout":     FieldKindDuration,
		"metadata":    FieldKindStruct,
		"nickname":    FieldKindWrapper,
		"contact":     FieldKindOneof,
		"age":         FieldKindScalar,
		"score":       FieldKindScalar,
		"level":       FieldKindScalar,
		"experience":  FieldKindScalar,
		"rating":      FieldKindScalar,
		"balance":     FieldKindScalar,
		"active":      FieldKindBool,
		"avatar":      FieldKindScalar,
		"update_mask": FieldKindMessage,
	}

	for path, wantKind := range expectedTopLevel {
		gotKind, ok := byPath[path]
		if !ok {
			t.Errorf("path %q not found in WalkMessage results", path)
			continue
		}
		if gotKind != wantKind {
			t.Errorf("path %q: got kind %v, want %v", path, gotKind, wantKind)
		}
	}

	// Verify nested address fields have dot-notation paths.
	nestedPaths := []string{
		"address.street",
		"address.city",
		"address.state",
		"address.zip_code",
		"address.country",
	}
	for _, p := range nestedPaths {
		if _, ok := byPath[p]; !ok {
			t.Errorf("nested path %q not found", p)
		}
	}

	// Verify oneof has peers.
	for _, info := range infos {
		if info.Kind == FieldKindOneof && info.Name == "contact" {
			if len(info.OneofPeers) != 2 {
				t.Errorf("oneof 'contact' should have 2 peers, got %d", len(info.OneofPeers))
			}
			peerNames := make(map[string]bool)
			for _, p := range info.OneofPeers {
				peerNames[p.Name] = true
			}
			if !peerNames["phone"] || !peerNames["slack_handle"] {
				t.Errorf("oneof 'contact' peers should include phone and slack_handle, got %v", peerNames)
			}
		}
	}

	// Verify enum field has enum values.
	for _, info := range infos {
		if info.Name == "role" && info.Kind == FieldKindEnum {
			if len(info.EnumValues) != 4 {
				t.Errorf("enum 'role' should have 4 values, got %d", len(info.EnumValues))
			}
		}
	}

	// Verify map field has key/value children.
	for _, info := range infos {
		if info.Name == "labels" && info.Kind == FieldKindMap {
			if len(info.Children) != 2 {
				t.Errorf("map 'labels' should have 2 children (key, value), got %d", len(info.Children))
			}
		}
	}
}

func TestIsWellKnownType(t *testing.T) {
	loader := newTestLoader()
	reg, _, err := loader.Load([]string{
		filepath.Join(testdataDir, "basic.proto"),
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	m, ok := reg.GetMethod("testdata.UserService", "CreateUser")
	if !ok {
		t.Fatal("CreateUser not found")
	}

	inputDesc := m.Desc.Input()

	wellKnownFields := []string{"created_at", "timeout", "metadata", "nickname"}
	for _, name := range wellKnownFields {
		fd := inputDesc.Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			t.Errorf("field %q not found", name)
			continue
		}
		if fd.Kind() != protoreflect.MessageKind {
			t.Errorf("field %q is not a message kind", name)
			continue
		}
		if !IsWellKnownType(fd.Message()) {
			t.Errorf("field %q should be a well-known type", name)
		}
	}

	// Address is NOT a well-known type.
	addressField := inputDesc.Fields().ByName("address")
	if addressField == nil {
		t.Fatal("address field not found")
	}
	if IsWellKnownType(addressField.Message()) {
		t.Error("Address should not be a well-known type")
	}
}

// Helper to recursively collect all paths from FieldInfo trees.
func collectPaths(infos []FieldInfo, m map[string]FieldKind) {
	for _, info := range infos {
		m[info.Path] = info.Kind
		if info.Kind == FieldKindMessage || info.Kind == FieldKindMap || info.Kind == FieldKindRepeated {
			collectPaths(info.Children, m)
		}
	}
}

func pkgNames(pkgs []PackageEntry) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = string(p.Name)
	}
	return names
}
