package explorer

import (
	tea "charm.land/bubbletea/v2"

	"github.com/futuramacoder/protopilot/internal/messages"
	"github.com/futuramacoder/protopilot/internal/proto"
)

// Model is the Bubble Tea model for the explorer pane.
type Model struct {
	roots   []*TreeNode
	visible []*TreeNode
	cursor  int
	focused bool
	width   int
	height  int
}

// New creates a new explorer model from a proto registry.
func New(reg *proto.Registry) Model {
	m := Model{}
	m.SetRegistry(reg)
	return m
}

// SetRegistry replaces the tree (used on proto reload).
func (m *Model) SetRegistry(reg *proto.Registry) {
	m.roots = BuildTree(reg)
	m.visible = FlattenVisible(m.roots)
	m.cursor = 0
}

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "g":
		m.cursor = 0
	case "G":
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
		}
	case "enter", "l", "right":
		return m.selectOrToggle()
	case "h", "left":
		m.collapseOrParent()
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.visible) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

func (m Model) selectOrToggle() (tea.Model, tea.Cmd) {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return m, nil
	}

	node := m.visible[m.cursor]

	switch node.Kind {
	case NodePackage, NodeService:
		node.Expanded = !node.Expanded
		m.visible = FlattenVisible(m.roots)
		// Clamp cursor after collapse.
		if m.cursor >= len(m.visible) {
			m.cursor = len(m.visible) - 1
		}
	case NodeMethod:
		// Find the parent service full name.
		svcFullName := findParentService(m.roots, node)
		return m, func() tea.Msg {
			return messages.MethodSelectedMsg{
				ServiceFullName: svcFullName,
				MethodName:      node.Label,
				MethodDesc:      node.MethodDesc,
				IsStreaming:     node.IsStreaming,
			}
		}
	}

	return m, nil
}

func (m *Model) collapseOrParent() {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return
	}

	node := m.visible[m.cursor]

	// If it's an expanded container, collapse it.
	if (node.Kind == NodePackage || node.Kind == NodeService) && node.Expanded {
		node.Expanded = false
		m.visible = FlattenVisible(m.roots)
		return
	}

	// Otherwise move to the parent node.
	parent := findParentNode(m.roots, node)
	if parent != nil {
		for i, v := range m.visible {
			if v == parent {
				m.cursor = i
				return
			}
		}
	}
}

// findParentService returns the full service name for a method node.
func findParentService(roots []*TreeNode, method *TreeNode) string {
	for _, pkg := range roots {
		for _, svc := range pkg.Children {
			for _, m := range svc.Children {
				if m == method {
					return svc.FullName
				}
			}
		}
	}
	return ""
}

// findParentNode returns the parent of the given node in the tree.
func findParentNode(roots []*TreeNode, target *TreeNode) *TreeNode {
	for _, root := range roots {
		if p := findParent(root, target); p != nil {
			return p
		}
	}
	return nil
}

func findParent(node *TreeNode, target *TreeNode) *TreeNode {
	for _, child := range node.Children {
		if child == target {
			return node
		}
		if p := findParent(child, target); p != nil {
			return p
		}
	}
	return nil
}
