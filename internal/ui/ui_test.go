package ui

import (
	"testing"

	"google.golang.org/grpc/codes"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		code codes.Code
		want string // "success", "warning", "error", "dimmed"
	}{
		{codes.OK, "success"},
		{codes.NotFound, "warning"},
		{codes.InvalidArgument, "warning"},
		{codes.AlreadyExists, "warning"},
		{codes.Canceled, "warning"},
		{codes.DeadlineExceeded, "warning"},
		{codes.Internal, "error"},
		{codes.Unavailable, "error"},
		{codes.Unauthenticated, "error"},
		{codes.PermissionDenied, "error"},
	}

	for _, tt := range tests {
		c := StatusColor(tt.code)
		if c == nil {
			t.Errorf("StatusColor(%v) returned nil", tt.code)
		}
	}

	// Verify specific mappings
	if StatusColor(codes.OK) != ColorSuccess {
		t.Error("OK should map to success color")
	}
	if StatusColor(codes.NotFound) != ColorWarning {
		t.Error("NotFound should map to warning color")
	}
	if StatusColor(codes.Internal) != ColorError {
		t.Error("Internal should map to error color")
	}
}

func TestStylesExist(t *testing.T) {
	// Verify all styles render without panic.
	styles := []struct {
		name   string
		render string
	}{
		{"PaneBorder", PaneBorder.Render("test")},
		{"PaneFocusedBorder", PaneFocusedBorder.Render("test")},
		{"TitleStyle", TitleStyle.Render("test")},
		{"DimmedStyle", DimmedStyle.Render("test")},
		{"ErrorStyle", ErrorStyle.Render("test")},
		{"SuccessStyle", SuccessStyle.Render("test")},
		{"WarningStyle", WarningStyle.Render("test")},
		{"SelectedStyle", SelectedStyle.Render("test")},
		{"HelpBarStyle", HelpBarStyle.Render("test")},
		{"SecondaryStyle", SecondaryStyle.Render("test")},
	}

	for _, s := range styles {
		if s.render == "" {
			t.Errorf("style %s rendered empty", s.name)
		}
	}
}

func TestHelpBar(t *testing.T) {
	bar := NewHelpBar()

	bindings := []KeyBinding{
		{Key: "Tab", Description: "switch pane"},
		{Key: "Ctrl+Enter", Description: "send"},
	}

	// Without warnings.
	view := bar.View(bindings, nil)
	if view == "" {
		t.Error("HelpBar.View returned empty")
	}

	// With warnings.
	view = bar.View(bindings, []string{"2 files failed to parse"})
	if view == "" {
		t.Error("HelpBar.View with warnings returned empty")
	}
}

func TestLogo(t *testing.T) {
	logo := Logo()
	if logo == "" {
		t.Error("Logo() returned empty")
	}
}

func TestEmptyState(t *testing.T) {
	state := EmptyState("Select a method from the explorer to begin", 80, 24)
	if state == "" {
		t.Error("EmptyState() returned empty")
	}
}
