package explorer

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/futuramacoder/protopilot/internal/proto"
)

// NodeKind identifies the type of tree node.
type NodeKind int

const (
	NodePackage NodeKind = iota
	NodeService
	NodeMethod
)

// TreeNode represents a single node in the explorer tree.
type TreeNode struct {
	Kind       NodeKind
	Label      string                        // Display label
	FullName   string                        // Fully qualified name
	Depth      int                           // Indent level (0=package, 1=service, 2=method)
	Expanded   bool                          // For package/service nodes
	IsStreaming bool                         // For method nodes
	MethodDesc protoreflect.MethodDescriptor // Non-nil for method nodes
	Children   []*TreeNode
}

// BuildTree creates the tree from a proto.Registry.
func BuildTree(reg *proto.Registry) []*TreeNode {
	if reg == nil {
		return nil
	}
	packages := reg.Packages()
	roots := make([]*TreeNode, 0, len(packages))

	for _, pkg := range packages {
		pkgNode := &TreeNode{
			Kind:     NodePackage,
			Label:    string(pkg.Name),
			FullName: string(pkg.Name),
			Depth:    0,
			Expanded: true,
		}

		for _, svc := range pkg.Services {
			// Use short name (without package prefix) for display.
			shortName := string(svc.Desc.Name())
			svcNode := &TreeNode{
				Kind:     NodeService,
				Label:    shortName,
				FullName: string(svc.Name),
				Depth:    1,
				Expanded: true,
			}

			for _, m := range svc.Methods {
				methodNode := &TreeNode{
					Kind:       NodeMethod,
					Label:      string(m.Name),
					FullName:   string(svc.Name) + "/" + string(m.Name),
					Depth:      2,
					IsStreaming: m.IsStreaming,
					MethodDesc: m.Desc,
				}
				svcNode.Children = append(svcNode.Children, methodNode)
			}

			pkgNode.Children = append(pkgNode.Children, svcNode)
		}

		roots = append(roots, pkgNode)
	}

	return roots
}

// FlattenVisible returns only the visible nodes (respecting expand/collapse
// state) as a flat slice for rendering and cursor navigation.
func FlattenVisible(roots []*TreeNode) []*TreeNode {
	var result []*TreeNode
	for _, root := range roots {
		flattenNode(root, &result)
	}
	return result
}

func flattenNode(node *TreeNode, result *[]*TreeNode) {
	*result = append(*result, node)
	if node.Expanded {
		for _, child := range node.Children {
			flattenNode(child, result)
		}
	}
}
