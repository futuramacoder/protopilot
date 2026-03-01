package proto_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestProtoFilesCompile verifies that all test proto files can be parsed
// by protocompile and that the resulting descriptors work with dynamicpb.
func TestProtoFilesCompile(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "proto", "testdata")

	// Verify testdata directory exists.
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Fatalf("testdata directory not found: %s", testdataDir)
	}

	protoFiles := []string{
		"basic.proto",
		"orders.proto",
		"separate_pkg.proto",
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{testdataDir},
		}),
		Reporter: reporter.NewReporter(nil, nil),
	}

	files, err := compiler.Compile(context.Background(), protoFiles...)
	if err != nil {
		t.Fatalf("failed to compile proto files: %v", err)
	}

	if len(files) != len(protoFiles) {
		t.Fatalf("expected %d compiled files, got %d", len(protoFiles), len(files))
	}

	// Verify basic.proto contents.
	basicFile := files.FindFileByPath("basic.proto")
	if basicFile == nil {
		t.Fatal("basic.proto not found in compiled files")
	}

	services := basicFile.Services()
	if services.Len() == 0 {
		t.Fatal("basic.proto has no services")
	}

	userSvc := services.ByName("UserService")
	if userSvc == nil {
		t.Fatal("UserService not found")
	}

	methods := userSvc.Methods()
	if methods.Len() != 3 {
		t.Fatalf("expected 3 methods in UserService, got %d", methods.Len())
	}

	// Verify GetUser method.
	getUser := methods.ByName("GetUser")
	if getUser == nil {
		t.Fatal("GetUser method not found")
	}
	if getUser.IsStreamingClient() || getUser.IsStreamingServer() {
		t.Error("GetUser should not be streaming")
	}

	// Verify ListUsers is server-streaming.
	listUsers := methods.ByName("ListUsers")
	if listUsers == nil {
		t.Fatal("ListUsers method not found")
	}
	if !listUsers.IsStreamingServer() {
		t.Error("ListUsers should be server-streaming")
	}

	// Verify orders.proto is in the same package.
	ordersFile := files.FindFileByPath("orders.proto")
	if ordersFile == nil {
		t.Fatal("orders.proto not found")
	}
	if ordersFile.Package() != basicFile.Package() {
		t.Errorf("orders.proto package %q != basic.proto package %q",
			ordersFile.Package(), basicFile.Package())
	}

	// Verify separate_pkg.proto is in a different package.
	sepFile := files.FindFileByPath("separate_pkg.proto")
	if sepFile == nil {
		t.Fatal("separate_pkg.proto not found")
	}
	if sepFile.Package() == basicFile.Package() {
		t.Error("separate_pkg.proto should be in a different package than basic.proto")
	}
	if sepFile.Package() != "payments" {
		t.Errorf("expected package 'payments', got %q", sepFile.Package())
	}

	// Smoke test: dynamicpb integration.
	// Create a dynamic message from CreateUserRequest and set a field.
	createUser := userSvc.Methods().ByName("CreateUser")
	if createUser == nil {
		t.Fatal("CreateUser method not found")
	}
	inputDesc := createUser.Input()
	msg := dynamicpb.NewMessage(inputDesc)

	nameField := inputDesc.Fields().ByName("name")
	if nameField == nil {
		t.Fatal("'name' field not found in CreateUserRequest")
	}

	msg.Set(nameField, protoreflect.ValueOfString("test-user"))
	got := msg.Get(nameField).String()
	if got != "test-user" {
		t.Errorf("expected 'test-user', got %q", got)
	}
}
