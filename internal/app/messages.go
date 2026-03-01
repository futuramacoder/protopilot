package app

import "github.com/futuramacoder/protopilot/internal/messages"

// Re-export message types from the messages package so app.go can use them directly
// and existing references in app package work.
type (
	MethodSelectedMsg  = messages.MethodSelectedMsg
	SendRequestMsg     = messages.SendRequestMsg
	ResponseReceivedMsg = messages.ResponseReceivedMsg
	FocusPaneMsg       = messages.FocusPaneMsg
	ProtoReloadMsg     = messages.ProtoReloadMsg
	ProtoLoadedMsg     = messages.ProtoLoadedMsg
	ConnectionChangedMsg = messages.ConnectionChangedMsg
	TerminalTooSmallMsg = messages.TerminalTooSmallMsg
	CopyGrpcurlMsg     = messages.CopyGrpcurlMsg
)

// Re-export PaneID type and constants.
type PaneID = messages.PaneID

const (
	PaneExplorer       = messages.PaneExplorer
	PaneRequestBuilder = messages.PaneRequestBuilder
	PaneResponseViewer = messages.PaneResponseViewer
)
