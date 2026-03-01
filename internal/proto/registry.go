package proto

import (
	"sort"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Registry holds all parsed proto descriptors, indexed for lookup.
type Registry struct {
	packages map[protoreflect.FullName]*PackageEntry
}

// PackageEntry represents a proto package with its services.
type PackageEntry struct {
	Name     protoreflect.FullName
	Services []ServiceEntry
}

// ServiceEntry represents a gRPC service with its methods.
type ServiceEntry struct {
	Name    protoreflect.FullName
	Desc    protoreflect.ServiceDescriptor
	Methods []MethodEntry
}

// MethodEntry represents a single RPC method.
type MethodEntry struct {
	Name        protoreflect.Name
	Desc        protoreflect.MethodDescriptor
	IsStreaming  bool // true if client or server streaming
}

const defaultPackageName protoreflect.FullName = "(default)"

// buildRegistry constructs a Registry from compiled file descriptors.
// Services in the same package from different files are merged under
// one PackageEntry.
func buildRegistry(files []protoreflect.FileDescriptor) *Registry {
	r := &Registry{
		packages: make(map[protoreflect.FullName]*PackageEntry),
	}

	seen := make(map[protoreflect.FullName]bool)

	for _, fd := range files {
		pkgName := fd.Package()
		if pkgName == "" {
			pkgName = defaultPackageName
		}

		pkg, ok := r.packages[pkgName]
		if !ok {
			pkg = &PackageEntry{Name: pkgName}
			r.packages[pkgName] = pkg
		}

		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			sd := services.Get(i)
			svcFullName := sd.FullName()

			// Skip services already added (e.g. from duplicate file descriptors).
			if seen[svcFullName] {
				continue
			}
			seen[svcFullName] = true

			se := ServiceEntry{
				Name: svcFullName,
				Desc: sd,
			}

			methods := sd.Methods()
			for j := 0; j < methods.Len(); j++ {
				md := methods.Get(j)
				se.Methods = append(se.Methods, MethodEntry{
					Name:       md.Name(),
					Desc:       md,
					IsStreaming: md.IsStreamingClient() || md.IsStreamingServer(),
				})
			}

			pkg.Services = append(pkg.Services, se)
		}
	}

	return r
}

// Packages returns all packages sorted alphabetically.
func (r *Registry) Packages() []PackageEntry {
	result := make([]PackageEntry, 0, len(r.packages))
	for _, pkg := range r.packages {
		if len(pkg.Services) > 0 {
			result = append(result, *pkg)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetMethod looks up a method by full service name and method name.
func (r *Registry) GetMethod(service protoreflect.FullName, method protoreflect.Name) (MethodEntry, bool) {
	for _, pkg := range r.packages {
		for _, svc := range pkg.Services {
			if svc.Name == service {
				for _, m := range svc.Methods {
					if m.Name == method {
						return m, true
					}
				}
				return MethodEntry{}, false
			}
		}
	}
	return MethodEntry{}, false
}
