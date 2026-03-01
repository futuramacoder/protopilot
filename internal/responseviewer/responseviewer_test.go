package responseviewer

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"

	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
	"github.com/futuramacoder/protopilot/internal/messages"
)

func TestFormatJSON_PrettyPrint(t *testing.T) {
	input := []byte(`{"name":"Alice","age":30,"active":true}`)
	result := FormatJSON(input)

	// Should contain indented output (newlines from pretty-print).
	if !strings.Contains(result, "\n") {
		t.Error("expected pretty-printed JSON with newlines")
	}
	// Should contain the values.
	if !strings.Contains(result, "Alice") {
		t.Error("expected JSON to contain 'Alice'")
	}
	if !strings.Contains(result, "30") {
		t.Error("expected JSON to contain '30'")
	}
}

func TestFormatJSON_SyntaxColors(t *testing.T) {
	input := []byte(`{"key":"value","num":42,"flag":true,"nil":null}`)
	result := FormatJSON(input)

	// ANSI escape codes should be present (color sequences start with \x1b[).
	if !strings.Contains(result, "\x1b[") {
		t.Error("expected ANSI color codes in output")
	}
	// Key, string, number, bool, null should all be present.
	if !strings.Contains(result, "key") {
		t.Error("expected 'key' in output")
	}
	if !strings.Contains(result, "value") {
		t.Error("expected 'value' in output")
	}
	if !strings.Contains(result, "42") {
		t.Error("expected '42' in output")
	}
	if !strings.Contains(result, "true") {
		t.Error("expected 'true' in output")
	}
	if !strings.Contains(result, "null") {
		t.Error("expected 'null' in output")
	}
}

func TestFormatJSON_Empty(t *testing.T) {
	result := FormatJSON(nil)
	if !strings.Contains(result, "empty") {
		t.Error("expected empty indicator for nil input")
	}
}

func TestFormatJSON_Invalid(t *testing.T) {
	input := []byte(`not json at all`)
	result := FormatJSON(input)
	if result != "not json at all" {
		t.Errorf("expected raw string for invalid JSON, got: %s", result)
	}
}

func TestFormatStatus_Colors(t *testing.T) {
	tests := []struct {
		code codes.Code
		want string
	}{
		{codes.OK, "OK"},
		{codes.NotFound, "NotFound"},
		{codes.InvalidArgument, "InvalidArgument"},
		{codes.Internal, "Internal"},
		{codes.Unavailable, "Unavailable"},
	}

	for _, tt := range tests {
		st := status.New(tt.code, "")
		result := FormatStatus(st)
		if !strings.Contains(result, tt.want) {
			t.Errorf("FormatStatus(%v) should contain %q, got: %s", tt.code, tt.want, result)
		}
		// Should contain ANSI codes.
		if !strings.Contains(result, "\x1b[") {
			t.Errorf("FormatStatus(%v) should contain color codes", tt.code)
		}
	}
}

func TestFormatStatus_Nil(t *testing.T) {
	result := FormatStatus(nil)
	if result != "" {
		t.Errorf("expected empty string for nil status, got: %s", result)
	}
}

func TestFormatStatus_WithMessage(t *testing.T) {
	st := status.New(codes.NotFound, "user not found")
	result := FormatStatus(st)
	if !strings.Contains(result, "user not found") {
		t.Errorf("expected status message in output, got: %s", result)
	}
}

func TestFormatLatency(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{500 * time.Microsecond, "500\u00B5s"},
		{12 * time.Millisecond, "12ms"},
		{150 * time.Millisecond, "150ms"},
		{1500 * time.Millisecond, "1.5s"},
		{3 * time.Second, "3.0s"},
	}

	for _, tt := range tests {
		result := FormatLatency(tt.duration)
		if !strings.Contains(result, tt.want) {
			t.Errorf("FormatLatency(%v) should contain %q, got: %s", tt.duration, tt.want, result)
		}
	}
}

func TestFormatErrorDetails(t *testing.T) {
	details := []grpcpkg.ErrorDetail{
		{
			Type:    "BadRequest",
			Content: "Field violations:\n  name: must not be empty",
		},
		{
			Type:    "DebugInfo",
			Content: "Detail: something went wrong",
		},
		{
			Type:    "RetryInfo",
			Content: "Retry after: 5s",
		},
	}

	result := FormatErrorDetails(details)

	if !strings.Contains(result, "Error Details") {
		t.Error("expected 'Error Details' header")
	}
	if !strings.Contains(result, "BadRequest") {
		t.Error("expected 'BadRequest' type")
	}
	if !strings.Contains(result, "must not be empty") {
		t.Error("expected field violation content")
	}
	if !strings.Contains(result, "DebugInfo") {
		t.Error("expected 'DebugInfo' type")
	}
	if !strings.Contains(result, "RetryInfo") {
		t.Error("expected 'RetryInfo' type")
	}
	if !strings.Contains(result, "5s") {
		t.Error("expected retry delay")
	}
}

func TestFormatErrorDetails_Empty(t *testing.T) {
	result := FormatErrorDetails(nil)
	if result != "" {
		t.Errorf("expected empty string for nil details, got: %s", result)
	}
}

func TestFormatMetadata(t *testing.T) {
	md := metadata.MD{
		"content-type": {"application/grpc"},
		"x-request-id": {"abc-123"},
	}

	result := FormatMetadata("Headers", md)

	if !strings.Contains(result, "Headers") {
		t.Error("expected 'Headers' label")
	}
	if !strings.Contains(result, "content-type") {
		t.Error("expected 'content-type' key")
	}
	if !strings.Contains(result, "application/grpc") {
		t.Error("expected 'application/grpc' value")
	}
	if !strings.Contains(result, "x-request-id") {
		t.Error("expected 'x-request-id' key")
	}
	if !strings.Contains(result, "abc-123") {
		t.Error("expected 'abc-123' value")
	}
}

func TestFormatMetadata_Empty(t *testing.T) {
	result := FormatMetadata("Headers", nil)
	if result != "" {
		t.Errorf("expected empty string for nil metadata, got: %s", result)
	}
}

func TestModel_New(t *testing.T) {
	m := New()
	if m.hasResp {
		t.Error("new model should not have response")
	}
	if m.loading {
		t.Error("new model should not be loading")
	}
	if m.focused {
		t.Error("new model should not be focused")
	}
}

func TestModel_SetLoading(t *testing.T) {
	m := New()
	m.SetLoading()
	if !m.loading {
		t.Error("expected loading to be true")
	}
	if m.hasResp {
		t.Error("expected hasResp to be false while loading")
	}
}

func TestModel_SetResponse(t *testing.T) {
	m := New()
	m.SetSize(80, 24)

	resp := messages.ResponseReceivedMsg{
		Body:    []byte(`{"name":"Alice"}`),
		Status:  status.New(codes.OK, ""),
		Latency: 42 * time.Millisecond,
		Headers: metadata.MD{
			"content-type": {"application/grpc"},
		},
	}

	m.SetResponse(resp)

	if m.loading {
		t.Error("expected loading to be false after SetResponse")
	}
	if !m.hasResp {
		t.Error("expected hasResp to be true after SetResponse")
	}
	if m.content == "" {
		t.Error("expected content to be set after SetResponse")
	}
	if !strings.Contains(m.content, "Alice") {
		t.Error("expected content to contain response body")
	}
	if !strings.Contains(m.content, "OK") {
		t.Error("expected content to contain status")
	}
	if !strings.Contains(m.content, "42ms") {
		t.Error("expected content to contain latency")
	}
}

func TestModel_SetResponse_Error(t *testing.T) {
	m := New()
	m.SetSize(80, 24)

	resp := messages.ResponseReceivedMsg{
		Status:  status.New(codes.NotFound, "user not found"),
		Latency: 10 * time.Millisecond,
		Err:     status.Error(codes.NotFound, "user not found"),
	}

	m.SetResponse(resp)

	if !m.hasResp {
		t.Error("expected hasResp to be true after error response")
	}
	if !strings.Contains(m.content, "user not found") {
		t.Error("expected content to contain error message")
	}
}

func TestModel_Clear(t *testing.T) {
	m := New()
	m.SetSize(80, 24)
	m.SetResponse(messages.ResponseReceivedMsg{
		Body:    []byte(`{"x":1}`),
		Status:  status.New(codes.OK, ""),
		Latency: time.Millisecond,
	})

	m.Clear()

	if m.hasResp {
		t.Error("expected hasResp to be false after Clear")
	}
	if m.loading {
		t.Error("expected loading to be false after Clear")
	}
	if m.content != "" {
		t.Error("expected content to be empty after Clear")
	}
}

func TestModel_ViewStates(t *testing.T) {
	m := New()
	m.SetSize(80, 24)

	// Empty state.
	view := m.View()
	if !strings.Contains(view.Content, "Send a request") {
		t.Error("empty view should show hint text")
	}

	// Loading state.
	m.SetLoading()
	view = m.View()
	if !strings.Contains(view.Content, "Sending request") {
		t.Error("loading view should show loading text")
	}

	// Response state.
	m.SetResponse(messages.ResponseReceivedMsg{
		Body:    []byte(`{"result":"ok"}`),
		Status:  status.New(codes.OK, ""),
		Latency: 5 * time.Millisecond,
	})
	view = m.View()
	if !strings.Contains(view.Content, "Response") {
		t.Error("response view should show Response title")
	}
}

func TestModel_Focus(t *testing.T) {
	m := New()
	m.SetFocused(true)
	if !m.focused {
		t.Error("expected focused to be true")
	}
	m.SetFocused(false)
	if m.focused {
		t.Error("expected focused to be false")
	}
}

func TestModel_SetSize(t *testing.T) {
	m := New()
	m.SetSize(100, 40)
	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestFormatJSON_NestedObject(t *testing.T) {
	input := []byte(`{"user":{"name":"Bob","address":{"city":"NYC"}}}`)
	result := FormatJSON(input)

	if !strings.Contains(result, "Bob") {
		t.Error("expected nested value 'Bob'")
	}
	if !strings.Contains(result, "NYC") {
		t.Error("expected deeply nested value 'NYC'")
	}
	// Should have multiple levels of indentation.
	if !strings.Contains(result, "    ") {
		t.Error("expected 4-space indentation for nesting")
	}
}

func TestFormatJSON_Array(t *testing.T) {
	input := []byte(`{"items":[1,2,3]}`)
	result := FormatJSON(input)

	if !strings.Contains(result, "1") {
		t.Error("expected array element '1'")
	}
	if !strings.Contains(result, "3") {
		t.Error("expected array element '3'")
	}
}
