package explorer

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/futuramacoder/protopilot/internal/messages"
	"github.com/futuramacoder/protopilot/internal/proto"
)

const testdataDir = "../../proto/testdata"

func loadTestRegistry(t *testing.T, files ...string) *proto.Registry {
	t.Helper()
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = filepath.Join(testdataDir, f)
	}
	loader := proto.NewLoader([]string{testdataDir})
	reg, _, err := loader.Load(paths)
	if err != nil {
		t.Fatalf("failed to load protos: %v", err)
	}
	return reg
}

func TestBuildTree(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto", "orders.proto", "separate_pkg.proto")
	roots := BuildTree(reg)

	if len(roots) != 2 {
		t.Fatalf("expected 2 package roots, got %d", len(roots))
	}

	// Packages should be sorted alphabetically.
	if roots[0].Label != "payments" {
		t.Errorf("expected first package 'payments', got %q", roots[0].Label)
	}
	if roots[1].Label != "testdata" {
		t.Errorf("expected second package 'testdata', got %q", roots[1].Label)
	}

	// testdata package should have 2 services.
	tdPkg := roots[1]
	if len(tdPkg.Children) != 2 {
		t.Fatalf("testdata should have 2 services, got %d", len(tdPkg.Children))
	}

	// Verify depths.
	for _, pkg := range roots {
		if pkg.Depth != 0 {
			t.Errorf("package depth should be 0, got %d", pkg.Depth)
		}
		for _, svc := range pkg.Children {
			if svc.Depth != 1 {
				t.Errorf("service depth should be 1, got %d", svc.Depth)
			}
			for _, m := range svc.Children {
				if m.Depth != 2 {
					t.Errorf("method depth should be 2, got %d", m.Depth)
				}
			}
		}
	}

	// Verify method nodes have descriptors.
	for _, pkg := range roots {
		for _, svc := range pkg.Children {
			for _, m := range svc.Children {
				if m.MethodDesc == nil {
					t.Errorf("method %q has nil descriptor", m.Label)
				}
			}
		}
	}
}

func TestFlattenVisible(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	roots := BuildTree(reg)

	// All expanded by default.
	visible := FlattenVisible(roots)
	// 1 package + 1 service + 3 methods = 5
	if len(visible) != 5 {
		t.Fatalf("expected 5 visible nodes (all expanded), got %d", len(visible))
	}

	// Collapse the service.
	svcNode := roots[0].Children[0]
	svcNode.Expanded = false
	visible = FlattenVisible(roots)
	// 1 package + 1 service (collapsed, no methods) = 2
	if len(visible) != 2 {
		t.Fatalf("expected 2 visible nodes (service collapsed), got %d", len(visible))
	}

	// Collapse the package.
	roots[0].Expanded = false
	visible = FlattenVisible(roots)
	// 1 package (collapsed) = 1
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible node (package collapsed), got %d", len(visible))
	}
}

func TestNavigation(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(true)
	m.SetSize(60, 30)

	// Initial cursor at 0.
	if m.cursor != 0 {
		t.Errorf("initial cursor should be 0, got %d", m.cursor)
	}

	// Move down with j.
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor after j should be 1, got %d", m.cursor)
	}

	// Move down again.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(Model)
	if m.cursor != 2 {
		t.Errorf("cursor after second j should be 2, got %d", m.cursor)
	}

	// Move up with k.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	m = updated.(Model)
	if m.cursor != 1 {
		t.Errorf("cursor after k should be 1, got %d", m.cursor)
	}

	// Jump to bottom with G.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'G', Text: "G"}))
	m = updated.(Model)
	if m.cursor != len(m.visible)-1 {
		t.Errorf("cursor after G should be %d, got %d", len(m.visible)-1, m.cursor)
	}

	// Jump to top with g.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'g', Text: "g"}))
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor after g should be 0, got %d", m.cursor)
	}

	// Cursor should not go below 0.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'k', Text: "k"}))
	m = updated.(Model)
	if m.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", m.cursor)
	}
}

func TestExpandCollapse(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(true)
	m.SetSize(60, 30)

	initialVisible := len(m.visible)

	// Cursor is on package node. Collapse with h.
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'h', Text: "h"}))
	m = updated.(Model)
	if len(m.visible) >= initialVisible {
		t.Error("collapsing package should reduce visible nodes")
	}
	if len(m.visible) != 1 {
		t.Errorf("after collapsing package, expected 1 visible, got %d", len(m.visible))
	}

	// Re-expand with Enter.
	updated, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: rune(tea.KeyEnter), Text: "enter"}))
	m = updated.(Model)
	if len(m.visible) != initialVisible {
		t.Errorf("after expanding package, expected %d visible, got %d", initialVisible, len(m.visible))
	}
}

func TestMethodSelection(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(true)
	m.SetSize(60, 30)

	// Navigate to first method (index 2: package=0, service=1, method=2).
	m.cursor = 2

	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: rune(tea.KeyEnter), Text: "enter"}))
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("expected a command from method selection")
	}

	msg := cmd()
	sel, ok := msg.(messages.MethodSelectedMsg)
	if !ok {
		t.Fatalf("expected MethodSelectedMsg, got %T", msg)
	}

	if sel.ServiceFullName != "testdata.UserService" {
		t.Errorf("expected service 'testdata.UserService', got %q", sel.ServiceFullName)
	}
	if sel.MethodDesc == nil {
		t.Error("MethodDesc should not be nil")
	}
}

func TestStreamingMethodDisplay(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(true)
	m.SetSize(80, 30)

	// Find the streaming method (ListUsers).
	var streamingNode *TreeNode
	for _, node := range m.visible {
		if node.Kind == NodeMethod && node.IsStreaming {
			streamingNode = node
			break
		}
	}

	if streamingNode == nil {
		t.Fatal("no streaming method found in tree")
	}

	if streamingNode.Label != "ListUsers" {
		t.Errorf("expected streaming method 'ListUsers', got %q", streamingNode.Label)
	}

	// Verify streaming tag appears in rendered view.
	view := m.View()
	if !strings.Contains(view.Content, "[stream]") {
		t.Error("view should contain [stream] tag for streaming methods")
	}
}

func TestStreamingMethodEmitsIsStreaming(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(true)
	m.SetSize(60, 30)

	// Find and select the streaming method.
	for i, node := range m.visible {
		if node.Kind == NodeMethod && node.IsStreaming {
			m.cursor = i
			break
		}
	}

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: rune(tea.KeyEnter), Text: "enter"}))
	if cmd == nil {
		t.Fatal("expected a command from streaming method selection")
	}

	msg := cmd()
	sel, ok := msg.(messages.MethodSelectedMsg)
	if !ok {
		t.Fatalf("expected MethodSelectedMsg, got %T", msg)
	}

	if !sel.IsStreaming {
		t.Error("IsStreaming should be true for streaming method")
	}
}

func TestUnfocusedIgnoresKeys(t *testing.T) {
	reg := loadTestRegistry(t, "basic.proto")
	m := New(reg)
	m.SetFocused(false)
	m.SetSize(60, 30)

	original := m.cursor
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Text: "j"}))
	m = updated.(Model)

	if m.cursor != original {
		t.Error("unfocused model should not respond to key events")
	}
}
