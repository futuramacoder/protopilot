package proto

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Loader wraps protocompile.Compiler for parsing .proto files.
type Loader struct {
	importPaths []string
}

// NewLoader creates a Loader with the given additional import paths.
// Well-known types are automatically available via WithStandardImports.
func NewLoader(importPaths []string) *Loader {
	return &Loader{importPaths: importPaths}
}

// Load parses the given .proto file paths and returns a Registry.
// Partial loading: files that fail to parse are skipped, and their
// errors are returned in the warnings slice. Only returns a non-nil
// error if ALL files fail.
func (l *Loader) Load(paths []string) (reg *Registry, warnings []string, err error) {
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no proto files specified")
	}

	// Collect unique directories from proto file paths and explicit import paths.
	importPaths := make([]string, 0, len(l.importPaths)+len(paths))
	importPaths = append(importPaths, l.importPaths...)

	seen := make(map[string]bool)
	for _, p := range l.importPaths {
		seen[p] = true
	}
	for _, p := range paths {
		dir := filepath.Dir(p)
		absDir, _ := filepath.Abs(dir)
		if !seen[absDir] {
			seen[absDir] = true
			importPaths = append(importPaths, absDir)
		}
	}

	// Compile each file independently to support partial loading.
	var fileDescs []protoreflect.FileDescriptor
	for _, p := range paths {
		base := filepath.Base(p)
		compiler := protocompile.Compiler{
			Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
				ImportPaths: importPaths,
			}),
			Reporter: reporter.NewReporter(nil, nil),
		}

		files, compileErr := compiler.Compile(context.Background(), base)
		if compileErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", p, compileErr))
			continue
		}

		for _, f := range files {
			fileDescs = append(fileDescs, f)
		}
	}

	if len(fileDescs) == 0 {
		return nil, warnings, fmt.Errorf("all proto files failed to parse")
	}

	reg = buildRegistry(fileDescs)
	return reg, warnings, nil
}
