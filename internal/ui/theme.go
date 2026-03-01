package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"google.golang.org/grpc/codes"
)

// Dark-only color palette constants.
var (
	ColorPrimary     = lipgloss.Color("#7C3AED")
	ColorSecondary   = lipgloss.Color("#06B6D4")
	ColorSuccess     = lipgloss.Color("#22C55E")
	ColorWarning     = lipgloss.Color("#EAB308")
	ColorError       = lipgloss.Color("#EF4444")
	ColorDimmed      = lipgloss.Color("#6B7280")
	ColorText        = lipgloss.Color("#F9FAFB")
	ColorBorder      = lipgloss.Color("#374151")
	ColorFocusBorder = lipgloss.Color("#7C3AED")
	ColorBackground  = lipgloss.Color("#111827")
)

// StatusColor returns the appropriate color for a gRPC status code.
func StatusColor(code codes.Code) color.Color {
	switch code {
	case codes.OK:
		return ColorSuccess
	case codes.NotFound, codes.InvalidArgument, codes.AlreadyExists,
		codes.FailedPrecondition, codes.OutOfRange:
		return ColorWarning
	case codes.Canceled, codes.DeadlineExceeded:
		return ColorWarning
	case codes.Internal, codes.Unavailable, codes.DataLoss,
		codes.Unknown, codes.Unimplemented, codes.ResourceExhausted,
		codes.Aborted, codes.PermissionDenied, codes.Unauthenticated:
		return ColorError
	default:
		return ColorDimmed
	}
}
