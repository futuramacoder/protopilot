package grpc

import (
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
)

// ErrorDetail represents a decoded gRPC error detail.
type ErrorDetail struct {
	Type    string // e.g., "BadRequest", "DebugInfo", "RetryInfo"
	Content string // Human-readable formatted content
	Raw     []byte // Original serialized bytes (for unknown types)
}

// DecodeErrorDetails extracts and formats error details from a gRPC status.
func DecodeErrorDetails(st *status.Status) []ErrorDetail {
	if st == nil {
		return nil
	}

	var details []ErrorDetail

	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.BadRequest:
			details = append(details, decodeBadRequest(v))
		case *errdetails.DebugInfo:
			details = append(details, decodeDebugInfo(v))
		case *errdetails.RetryInfo:
			details = append(details, decodeRetryInfo(v))
		default:
			// Try to decode as Any for unknown types.
			if anyMsg, ok := d.(*anypb.Any); ok {
				details = append(details, ErrorDetail{
					Type:    anyMsg.TypeUrl,
					Content: fmt.Sprintf("Unknown type: %s", anyMsg.TypeUrl),
					Raw:     anyMsg.Value,
				})
			} else {
				// Fallback: serialize to raw bytes.
				if pm, ok := d.(proto.Message); ok {
					raw, _ := proto.Marshal(pm)
					details = append(details, ErrorDetail{
						Type:    fmt.Sprintf("%T", d),
						Content: fmt.Sprintf("Unrecognized detail type: %T", d),
						Raw:     raw,
					})
				}
			}
		}
	}

	return details
}

func decodeBadRequest(br *errdetails.BadRequest) ErrorDetail {
	var lines []string
	for _, v := range br.FieldViolations {
		lines = append(lines, fmt.Sprintf("  %s: %s", v.Field, v.Description))
	}
	return ErrorDetail{
		Type:    "BadRequest",
		Content: "Field violations:\n" + strings.Join(lines, "\n"),
	}
}

func decodeDebugInfo(di *errdetails.DebugInfo) ErrorDetail {
	var parts []string
	if di.Detail != "" {
		parts = append(parts, "Detail: "+di.Detail)
	}
	if len(di.StackEntries) > 0 {
		parts = append(parts, "Stack trace:\n  "+strings.Join(di.StackEntries, "\n  "))
	}
	return ErrorDetail{
		Type:    "DebugInfo",
		Content: strings.Join(parts, "\n"),
	}
}

func decodeRetryInfo(ri *errdetails.RetryInfo) ErrorDetail {
	delay := "unknown"
	if ri.RetryDelay != nil {
		d := ri.RetryDelay.AsDuration()
		delay = d.String()
	}
	return ErrorDetail{
		Type:    "RetryInfo",
		Content: fmt.Sprintf("Retry after: %s", delay),
	}
}

// FormatRawDetail formats raw bytes as base64 for display.
func FormatRawDetail(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}
