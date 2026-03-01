package messages

import (
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/futuramacoder/protopilot/internal/proto"
)

// MethodSelectedMsg is sent when the user selects a method in the explorer.
type MethodSelectedMsg struct {
	ServiceFullName string
	MethodName      string
	MethodDesc      protoreflect.MethodDescriptor
	IsStreaming     bool
}

// SendRequestMsg is sent when the user presses Ctrl+Enter.
type SendRequestMsg struct {
	MethodDesc  protoreflect.MethodDescriptor
	FieldValues map[string]any
	Metadata    map[string]string
}

// ResponseReceivedMsg is sent when the gRPC call completes.
type ResponseReceivedMsg struct {
	Body     []byte // JSON-marshaled response
	Status   *status.Status
	Latency  time.Duration
	Headers  metadata.MD
	Trailers metadata.MD
	Err      error
}

// FocusPaneMsg cycles focus between panes.
type FocusPaneMsg struct {
	Pane PaneID
}

// PaneID identifies which pane is focused.
type PaneID int

const (
	PaneExplorer PaneID = iota
	PaneRequestBuilder
	PaneResponseViewer
)

// ProtoReloadMsg triggers re-parsing of all proto files.
type ProtoReloadMsg struct{}

// ProtoLoadedMsg is sent after proto parsing completes.
type ProtoLoadedMsg struct {
	Registry *proto.Registry
	Warnings []string
	Err      error
}

// ConnectionChangedMsg is sent when gRPC connection state changes.
type ConnectionChangedMsg struct {
	Host      string
	Connected bool
	Err       error
}

// TerminalTooSmallMsg is sent when the terminal is below 120x30.
type TerminalTooSmallMsg struct {
	Width  int
	Height int
}

// CopyGrpcurlMsg triggers copying the current request as a grpcurl command.
type CopyGrpcurlMsg struct{}
