package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
	"github.com/futuramacoder/protopilot/internal/ui"
)

func newTestModel() Model {
	cfg := Config{
		ProtoPaths: []string{"test.proto"},
		Host:       "localhost:50051",
		TLS:        grpcpkg.TLSConfig{Plaintext: true},
	}
	m := New(cfg)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return result.(Model)
}

func TestComputeLayout(t *testing.T) {
	l := ComputeLayout(120, 30)

	if l.ExplorerWidth != 36 {
		t.Errorf("expected explorer width 36, got %d", l.ExplorerWidth)
	}
	if l.RequestBuilderWidth != 84 {
		t.Errorf("expected request builder width 84, got %d", l.RequestBuilderWidth)
	}
	if l.ResponseViewerWidth != 84 {
		t.Errorf("expected response viewer width 84, got %d", l.ResponseViewerWidth)
	}
	if l.HelpBarHeight != 1 {
		t.Errorf("expected help bar height 1, got %d", l.HelpBarHeight)
	}

	usableHeight := 30 - 1
	if l.ExplorerHeight != usableHeight {
		t.Errorf("expected explorer height %d, got %d", usableHeight, l.ExplorerHeight)
	}
	if l.RequestBuilderHeight+l.ResponseViewerHeight != usableHeight {
		t.Errorf("right pane heights should sum to %d, got %d + %d",
			usableHeight, l.RequestBuilderHeight, l.ResponseViewerHeight)
	}
}

func TestComputeLayout_LargeTerminal(t *testing.T) {
	l := ComputeLayout(200, 50)

	if l.ExplorerWidth != 60 {
		t.Errorf("expected explorer width 60, got %d", l.ExplorerWidth)
	}
	if l.RequestBuilderWidth != 140 {
		t.Errorf("expected request builder width 140, got %d", l.RequestBuilderWidth)
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		ProtoPaths: []string{"test.proto"},
		Host:       "localhost:50051",
		TLS:        grpcpkg.TLSConfig{Plaintext: true},
	}
	m := New(cfg)

	if m.focus != PaneExplorer {
		t.Error("initial focus should be on explorer")
	}
	if m.tooSmall {
		t.Error("should not start as too small")
	}
	if m.helpVisible {
		t.Error("help should not be visible initially")
	}
	if m.warningsModal {
		t.Error("warnings modal should not be visible initially")
	}
	if m.hostModal.Visible() {
		t.Error("host modal should not be visible initially")
	}
}

func TestModel_WindowSize_TooSmall(t *testing.T) {
	cfg := Config{
		ProtoPaths: []string{"test.proto"},
		Host:       "localhost:50051",
		TLS:        grpcpkg.TLSConfig{Plaintext: true},
	}
	m := New(cfg)

	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	model := result.(Model)
	if !model.tooSmall {
		t.Error("expected tooSmall to be true for 80x20")
	}

	view := model.View()
	if view.Content == "" {
		t.Error("expected non-empty view when too small")
	}
}

func TestModel_WindowSize_OK(t *testing.T) {
	m := newTestModel()
	if m.tooSmall {
		t.Error("expected tooSmall to be false for 120x30")
	}
	if m.layout.ExplorerWidth == 0 {
		t.Error("expected layout to be computed")
	}
}

func TestModel_FocusCycling(t *testing.T) {
	m := newTestModel()

	if m.focus != PaneExplorer {
		t.Fatal("expected initial focus on explorer")
	}

	// Tab → request builder.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "tab"}))
	m = result.(Model)
	if m.focus != PaneRequestBuilder {
		t.Errorf("expected focus on request builder after tab, got %d", m.focus)
	}

	// Tab → response viewer.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "tab"}))
	m = result.(Model)
	if m.focus != PaneResponseViewer {
		t.Errorf("expected focus on response viewer after tab, got %d", m.focus)
	}

	// Tab → back to explorer.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "tab"}))
	m = result.(Model)
	if m.focus != PaneExplorer {
		t.Errorf("expected focus back on explorer after tab, got %d", m.focus)
	}

	// Shift+Tab → response viewer.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "shift+tab"}))
	m = result.(Model)
	if m.focus != PaneResponseViewer {
		t.Errorf("expected focus on response viewer after shift+tab, got %d", m.focus)
	}
}

func TestModel_QuitFromExplorer(t *testing.T) {
	m := newTestModel()

	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "q"}))
	if cmd == nil {
		t.Error("expected quit command from q key in explorer")
	}
}

func TestModel_HelpToggle(t *testing.T) {
	m := newTestModel()

	if m.helpVisible {
		t.Error("help should be hidden initially")
	}

	// ? opens help.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "?"}))
	m = result.(Model)
	if !m.helpVisible {
		t.Error("help should be visible after ? press")
	}

	// View should render overlay.
	view := m.View()
	if !strings.Contains(view.Content, "Keybindings") {
		t.Error("help overlay should contain 'Keybindings'")
	}

	// ? closes help.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "?"}))
	m = result.(Model)
	if m.helpVisible {
		t.Error("help should be hidden after second ? press")
	}

	// Esc also closes help.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "?"}))
	m = result.(Model)
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "escape"}))
	m = result.(Model)
	if m.helpVisible {
		t.Error("help should be hidden after Esc")
	}
}

func TestModel_WarningsModal(t *testing.T) {
	m := newTestModel()
	m.warnings = []string{"test.proto: syntax error at line 5"}

	// Ctrl+W opens warnings.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+w"}))
	m = result.(Model)
	if !m.warningsModal {
		t.Error("expected warnings modal to be visible")
	}

	// View should show warnings overlay.
	view := m.View()
	if !strings.Contains(view.Content, "Parse Warnings") {
		t.Error("warnings overlay should contain 'Parse Warnings'")
	}

	// Esc closes it.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "escape"}))
	m = result.(Model)
	if m.warningsModal {
		t.Error("expected warnings modal to close on Esc")
	}

	// Ctrl+W also toggles.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+w"}))
	m = result.(Model)
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+w"}))
	m = result.(Model)
	if m.warningsModal {
		t.Error("expected warnings modal to close on second Ctrl+W")
	}
}

func TestModel_HostModal(t *testing.T) {
	m := newTestModel()

	// Ctrl+H opens host modal.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+h"}))
	m = result.(Model)
	if !m.hostModal.Visible() {
		t.Error("expected host modal to be visible")
	}

	// View should show modal.
	view := m.View()
	if !strings.Contains(view.Content, "Change gRPC Host") {
		t.Error("host modal should contain 'Change gRPC Host'")
	}

	// Esc closes it.
	result, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "escape"}))
	m = result.(Model)
	if m.hostModal.Visible() {
		t.Error("expected host modal to close on Esc")
	}
}

func TestModel_ClipboardNoMethod(t *testing.T) {
	m := newTestModel()

	// Ctrl+Y with no method selected.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+y"}))
	m = result.(Model)
	if m.statusMsg != "No method selected" {
		t.Errorf("expected 'No method selected' status, got: %s", m.statusMsg)
	}
}

func TestModel_StatusMsgClearsOnKeyPress(t *testing.T) {
	m := newTestModel()
	m.statusMsg = "some status"

	// Any key press should clear status.
	result, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "tab"}))
	m = result.(Model)
	if m.statusMsg != "" {
		t.Error("expected status message to be cleared on key press")
	}
}

func TestDefaultHelpBindings(t *testing.T) {
	bindings := DefaultHelpBindings()
	if len(bindings) == 0 {
		t.Error("expected non-empty help bindings")
	}

	found := false
	for _, b := range bindings {
		if b.Key == "Tab" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Tab in help bindings")
	}
}

func TestHelpOverlay_Content(t *testing.T) {
	content := ui.HelpOverlay(120, 30)

	checks := []string{
		"Global", "Explorer", "Request Builder", "Response Viewer",
		"Tab", "Ctrl+Enter", "Ctrl+C", "Ctrl+R", "Ctrl+Y", "Ctrl+H", "F5",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("help overlay should contain %q", check)
		}
	}
}

func TestWarningsOverlay_Empty(t *testing.T) {
	content := ui.WarningsOverlay(nil, 120, 30)
	if !strings.Contains(content, "No warnings") {
		t.Error("empty warnings overlay should show 'No warnings'")
	}
}

func TestWarningsOverlay_WithWarnings(t *testing.T) {
	warnings := []string{"file1.proto: error", "file2.proto: error"}
	content := ui.WarningsOverlay(warnings, 120, 30)
	if !strings.Contains(content, "file1.proto") {
		t.Error("warnings overlay should contain first warning")
	}
	if !strings.Contains(content, "file2.proto") {
		t.Error("warnings overlay should contain second warning")
	}
}

func TestHostModal(t *testing.T) {
	modal := NewHostModal()

	if modal.Visible() {
		t.Error("modal should start hidden")
	}

	modal.Show("localhost:50051")
	if !modal.Visible() {
		t.Error("modal should be visible after Show")
	}

	// View should render.
	view := modal.View(120, 30)
	if !strings.Contains(view, "Change gRPC Host") {
		t.Error("modal view should contain title")
	}

	modal.Hide()
	if modal.Visible() {
		t.Error("modal should be hidden after Hide")
	}
}

func TestClipboardResultMsg(t *testing.T) {
	m := newTestModel()

	// Success.
	result, _ := m.Update(ClipboardResultMsg{Success: true})
	m = result.(Model)
	if m.statusMsg != "Copied grpcurl command!" {
		t.Errorf("expected success status, got: %s", m.statusMsg)
	}

	// Failure.
	result, _ = m.Update(ClipboardResultMsg{Success: false, Err: fmt.Errorf("no clipboard")})
	m = result.(Model)
	if !strings.Contains(m.statusMsg, "Clipboard error") {
		t.Errorf("expected error status, got: %s", m.statusMsg)
	}
}
