# Protopilot — Full Specification

> Definitive implementation reference. All design decisions from the interview are captured here.
> When this document conflicts with `architecture.md` or `CLAUDE.md`, **this document wins**.

---

## Table of Contents

1. [Design Decisions](#1-design-decisions)
2. [Technical Architecture](#2-technical-architecture)
3. [File-by-File Specification](#3-file-by-file-specification)
4. [Implementation Phases](#4-implementation-phases)
5. [Risks & Complexity Hotspots](#5-risks--complexity-hotspots)
6. [CLI Interface](#6-cli-interface)
7. [Keybinding Reference](#7-keybinding-reference)

---

## 1. Design Decisions

Every decision made during the 7-round interview, with rationale.

### 1.1 Proto Imports

**Decision:** Auto-resolve well-known types + explicit `--import-path` flag.

Well-known types (`google/protobuf/timestamp.proto`, etc.) are resolved automatically via `protocompile.WithStandardImports`. Users pass `--import-path` (repeatable) for custom import directories, similar to `protoc -I`. This gives zero-config for common cases while supporting complex proto layouts.

### 1.2 Streaming RPCs

**Decision:** Show grayed out in tree, block invocation.

Streaming methods (server-stream, client-stream, bidi) appear in the explorer tree but are visually dimmed with a `streaming` tag. Selecting one shows an informational message instead of generating a request form. This keeps the tree complete for discovery while clearly communicating what's supported.

### 1.3 Oneof Fields

**Decision:** Radio-group selector.

When a message contains a `oneof`, the form shows a selector to pick which variant is active. Only the selected variant's fields are displayed. Switching variants clears the previously selected fields. This enforces the proto constraint that exactly one of N fields can be set.

### 1.4 Repeated Fields

**Decision:** Collapsible sections, collapsed by default.

Each repeated field renders as a collapsible section showing the item count (e.g., `items (3)`). Expand to see/edit individual entries. Each entry has an add/remove button. Nested repeated messages get nested collapsible sections. This keeps the form compact for messages with many repeated fields.

### 1.5 Map Fields

**Decision:** Key-value rows with add/remove.

Each map entry is a row with a key input and a value input (or nested message fields if the value type is a message). Add/remove controls work like repeated fields. Keys are validated for uniqueness.

### 1.6 Error Details

**Decision:** Decode well-known error detail types; raw Any for unknown.

When a gRPC call returns a non-OK status with `google.rpc.Status.details`, decode known types:
- `google.rpc.BadRequest` — show field violations
- `google.rpc.DebugInfo` — show stack trace and detail
- `google.rpc.RetryInfo` — show retry delay

Unknown `Any` types are displayed as raw type URL + serialized bytes.

### 1.7 Connection Management

**Decision:** Persistent connection + manual reconnect keybinding + runtime host change.

The app maintains a persistent gRPC connection. A keybinding (`F5`) triggers disconnect/reconnect. A separate mechanism allows changing the host at runtime (modal input). Connection state is reflected in the status bar.

### 1.8 Multi-File Tree Organization

**Decision:** Merge services from same package under one node.

When multiple `.proto` files define services in the same package, those services are merged under a single package node in the explorer tree. Files with no package declaration are grouped under a `(default)` node.

### 1.9 Well-Known Types

**Decision:** Special widgets for common types.

| Proto Type | Widget | Input Format |
|---|---|---|
| `google.protobuf.Timestamp` | Datetime input | `2024-01-15T10:30:00Z` |
| `google.protobuf.Duration` | Duration input | `5s`, `1m30s`, `500ms` |
| `google.protobuf.Struct` | JSON text editor | Raw JSON object |
| `google.protobuf.Value` | JSON text editor | Any JSON value |
| `google.protobuf.StringValue` et al. | Nullable scalar | Text input with "null" option |
| `google.protobuf.FieldMask` | Text input | Comma-separated paths |

These types parse on submit via the codec layer. Invalid formats show a validation error.

### 1.10 Deep Nesting

**Decision:** Collapsible sections per nested message.

Each nested message is a collapsible group with a header showing the field name. Expand/collapse with Enter. Indentation visually indicates depth. Keeps the form compact — users only see the nesting level they're editing.

### 1.11 TLS Configuration

**Decision:** Full TLS config via CLI flags.

| Flag | Purpose |
|---|---|
| `--plaintext` | Disable TLS entirely |
| `--cacert` | CA certificate file path |
| `--cert` | Client certificate file for mTLS |
| `--key` | Client private key file for mTLS |
| `--servername` | TLS server name override (SNI) |

Covers mTLS, custom CAs, and enterprise setups. When no TLS flags are provided and `--plaintext` is not set, use system TLS defaults.

### 1.12 Empty States

**Decision:** ASCII art logo + usage hints.

When no method is selected, the request builder and response viewer show an ASCII art rendition of the Protopilot name plus contextual usage hints (e.g., "Select a method from the explorer to begin", "Press ? for help"). This creates a polished first impression.

### 1.13 Metadata (gRPC Headers)

**Decision:** Include in MVP as a collapsible section in the request builder.

A `Metadata` collapsible section sits at the top of the request builder, above the message fields. It contains key-value pair inputs with add/remove controls. Metadata is sent as gRPC headers on invocation. This is critical for auth-gated services.

### 1.14 Response Metadata Display

**Decision:** All-in-one view with dividers.

The response viewer shows everything in a single scrollable view:
1. **Response headers** (above body, with divider)
2. **JSON body** (main content)
3. **Response trailers** (below body, with divider)
4. **gRPC status + latency** (footer)

No tabs — everything visible at once.

### 1.15 Large Responses

**Decision:** Virtual scroll viewport.

The response viewer uses a `bubbles/viewport` that only renders visible lines. Supports smooth scrolling through arbitrarily large JSON responses. No truncation.

### 1.16 Enum Input

**Decision:** Popup overlay list.

When an enum field is focused and Enter is pressed, a small overlay list appears showing all enum values. The user navigates with j/k or arrows, selects with Enter, or dismisses with Escape. This handles enums with many values better than cycling.

### 1.17 Parse Errors

**Decision:** Partial load + status bar warning + keybinding for details.

When some `.proto` files fail to parse, the app loads what it can. A warning appears in the status/help bar showing the count of failed files (e.g., "2 files failed to parse"). A keybinding shows the full error details in a modal/overlay.

### 1.18 Validation

**Decision:** Live inline validation + validate all on submit.

As the user types, fields with invalid input show a red border and a brief inline error message (e.g., "not a valid int32"). On Ctrl+Enter, all fields are validated before sending. If any fail, the cursor moves to the first invalid field. Validation covers:
- Type constraints (numeric ranges, valid enum values)
- Well-known type format (timestamp, duration)
- Required fields (proto2)

### 1.19 Default Values

**Decision:** Empty fields with grayed placeholder text.

Fields start empty (not pre-filled). Grayed placeholder text shows the type and proto3 default (e.g., `int32, default: 0`, `string`, `bool, default: false`). Submitting an empty field means "use default" / "field not set" in proto3 semantics.

### 1.20 Color Theme

**Decision:** Dark only.

Single dark theme. No light mode, no user configuration. Covers the vast majority of terminal users and ensures consistent appearance in documentation and screenshots.

**Color Palette:**

| Token | Hex | Usage |
|---|---|---|
| Primary | `#7C3AED` | Purple accent, focused borders |
| Secondary | `#06B6D4` | Cyan highlights |
| Success | `#22C55E` | gRPC OK status |
| Warning | `#EAB308` | NOT_FOUND, INVALID_ARGUMENT |
| Error | `#EF4444` | INTERNAL, UNAVAILABLE, validation errors |
| Dimmed | `#6B7280` | Placeholders, disabled items, streaming RPCs |
| Text | `#F9FAFB` | Primary text |
| Border | `#374151` | Unfocused pane borders |
| FocusBorder | `#7C3AED` | Focused pane border (same as Primary) |
| Background | `#111827` | Dark background |

### 1.21 Keyboard Navigation

**Decision:** Both Vim keys (j/k/h/l) and arrow keys.

All navigation supports both styles. Vim users feel at home; others aren't alienated. This is standard for modern TUI applications.

### 1.22 Hot Reload

**Decision:** Manual Ctrl+R to re-parse protos.

Pressing `Ctrl+R` re-parses all proto files from their original paths. A status message confirms success or shows errors. No file watchers — explicit, predictable, no background I/O.

### 1.23 Copy as grpcurl

**Decision:** Keybinding copies current request as a grpcurl command.

Pressing `Ctrl+Y` constructs a `grpcurl` command from the current method, field values, metadata, and connection settings, then copies it to the system clipboard. Example output:

```bash
grpcurl -plaintext -d '{"name":"world"}' \
  -H 'authorization: Bearer token123' \
  localhost:50051 helloworld.Greeter/SayHello
```

Clipboard integration: `xclip` (Linux), `pbcopy` (macOS), `clip.exe` (Windows/WSL).

### 1.24 Proto Parsing Library

**Decision:** `bufbuild/protocompile` replaces `jhump/protoreflect`.

The Buf team's `protocompile` is newer, better maintained, and handles edge cases better. It returns standard `protoreflect.FileDescriptor` types (from `google.golang.org/protobuf`), which integrate directly with `dynamicpb.NewMessage`. This is a departure from `architecture.md`, which references `jhump/protoreflect`.

### 1.25 Bubble Tea Version

**Decision:** Bubble Tea v2.

Use v2 as specified in `architecture.md`. Newer API with better layout primitives. Note: v2 uses `charm.land` vanity import paths instead of `github.com/charmbracelet`.

### 1.26 Minimum Terminal Size

**Decision:** 120x30 (modern).

The three-pane layout requires reasonable width. If the terminal is smaller than 120x30, show a centered message asking the user to resize. No degraded layout mode.

### 1.27 Test Data

**Decision:** Generic test protos covering all field types.

Create representative test `.proto` files covering: simple CRUD, nested messages, enums, repeated fields, maps, oneof, well-known types, multiple services in one package, and services across packages.

---

## 2. Technical Architecture

### 2.1 Dependencies

```
module github.com/futuramacoder/protopilot

go 1.26.0

require (
    charm.land/x/bubbletea/v2         // TUI framework (Elm architecture)
    charm.land/x/lipgloss/v2          // Terminal styling
    charm.land/x/bubbles/v2           // Reusable TUI components (viewport, textinput)
    github.com/bufbuild/protocompile  // Runtime .proto parsing (no protoc needed)
    google.golang.org/protobuf        // dynamicpb, protoreflect, protojson
    google.golang.org/grpc            // gRPC client
    google.golang.org/genproto        // errdetails (BadRequest, DebugInfo, RetryInfo)
    github.com/spf13/cobra            // CLI flag parsing
    github.com/stretchr/testify       // Test assertions
)
```

**Key difference from architecture.md:** `bufbuild/protocompile` replaces `jhump/protoreflect`. Charm libraries use `charm.land` vanity imports for v2.

### 2.2 Cross-Pane Message Types

```go
package app

import (
    "time"

    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/reflect/protoreflect"
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
    Body     []byte        // JSON-marshaled response
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
```

### 2.3 Data Flow

```
1. Startup
   CLI args
     ├── proto paths + import paths → proto.Loader (protocompile.Compiler)
     │     → proto.Registry (merged by package)
     │           → Explorer builds tree
     └── host + TLS flags → grpc.Client (persistent connection)

2. Method Selection
   Explorer ──MethodSelectedMsg──→ App ──→ RequestBuilder
   RequestBuilder ← skeleton.Generate(method.Input())
   RequestBuilder shows form + Metadata section

3. Request Submission
   RequestBuilder ──SendRequestMsg──→ App
   App → grpc.Codec.BuildMessage(fieldValues, descriptor) → dynamicpb.Message
   App → grpc.Invoker.InvokeUnary(conn, method, message, metadata) → tea.Cmd

4. Response
   Invoker ──ResponseReceivedMsg──→ App ──→ ResponseViewer
   ResponseViewer ← formatter.FormatJSON(body)
                   + formatter.FormatMetadata(headers, trailers)
                   + formatter.FormatStatus(status)
                   + formatter.FormatErrorDetails(status.Details())
                   + formatter.FormatLatency(latency)

5. Hot Reload (Ctrl+R)
   App ──ProtoReloadMsg──→ proto.Loader re-parses
     ──ProtoLoadedMsg──→ App updates Registry → Explorer rebuilds tree

6. Connection Management (F5 / host change)
   App → grpc.Client.Reconnect() or grpc.Client.ChangeHost(newHost)
     ──ConnectionChangedMsg──→ App updates status bar
```

### 2.4 Field Classification System

The request builder classifies every proto field into one of 11 `FieldKind` values. Each kind determines the widget type, validation rules, and serialization behavior.

```go
type FieldKind int

const (
    FieldKindScalar    FieldKind = iota // string, int32, int64, float, double, bytes
    FieldKindBool                       // bool → toggle widget
    FieldKindEnum                       // enum → popup list widget
    FieldKindMessage                    // nested message → collapsible section
    FieldKindRepeated                   // repeated field → collapsible list
    FieldKindMap                        // map field → key-value rows
    FieldKindOneof                      // oneof group → radio selector
    FieldKindTimestamp                  // google.protobuf.Timestamp → datetime input
    FieldKindDuration                   // google.protobuf.Duration → duration input
    FieldKindStruct                     // google.protobuf.Struct/Value → JSON editor
    FieldKindWrapper                    // google.protobuf.StringValue etc. → nullable input
)
```

Classification priority (highest first):
1. Well-known types (Timestamp, Duration, Struct, Value, wrappers) → specific kind
2. Map → `FieldKindMap`
3. Repeated → `FieldKindRepeated`
4. Oneof → `FieldKindOneof` (for the group; inner fields classified normally)
5. Message → `FieldKindMessage`
6. Enum → `FieldKindEnum`
7. Bool → `FieldKindBool`
8. Everything else → `FieldKindScalar`

### 2.5 Layout

```
┌─────────────────────────────────────────────────────────────────┐
│                         Protopilot                              │
├──────────────┬──────────────────────────────────────────────────┤
│              │ Request Builder                                  │
│  Explorer    │ ┌ Metadata ──────────────────────────────────┐  │
│              │ │ authorization: Bearer xxx                  │  │
│  📦 package  │ └────────────────────────────────────────────┘  │
│  ├─ Service  │ ┌ Fields ────────────────────────────────────┐  │
│  │  ├─ Meth  │ │ name: [________]                          │  │
│  │  └─ Meth  │ │ ▸ address (message)                       │  │
│  └─ Service  │ │ tags (3): [▸]                             │  │
│              │ └────────────────────────────────────────────┘  │
│   ~30%       ├──────────────────────────────────────────────────┤
│              │ Response Viewer                                  │
│              │ ── Headers ──────────────────────────────────── │
│              │ content-type: application/grpc                   │
│              │ ── Body ─────────────────────────────────────── │
│              │ {                                                │
│              │   "message": "Hello, world!"                    │
│              │ }                                                │
│              │ ── Trailers ─────────────────────────────────── │
│              │ ✓ OK  12ms                                      │
├──────────────┴──────────────────────────────────────────────────┤
│ Tab: switch pane │ Ctrl+Enter: send │ ?: help │ 2 files failed │
└─────────────────────────────────────────────────────────────────┘
```

- Explorer: ~30% width
- Right panes: ~70% width, split vertically 50/50
- Help bar: bottom row, context-sensitive shortcuts + status warnings
- Minimum terminal: 120 columns x 30 rows

---

## 3. File-by-File Specification

### 3.1 `proto/testdata/` — Test Proto Files

#### `proto/testdata/basic.proto`

Comprehensive test proto covering all field types for a single service.

```protobuf
syntax = "proto3";
package testdata;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/wrappers.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc ListUsers(ListUsersRequest) returns (stream ListUsersResponse); // streaming
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  User user = 1;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  UserRole role = 3;
  Address address = 4;
  repeated string tags = 5;
  map<string, string> labels = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Duration timeout = 8;
  google.protobuf.Struct metadata = 9;
  google.protobuf.StringValue nickname = 10;

  oneof contact {
    string phone = 11;
    string slack_handle = 12;
  }
}

// ... enums, nested messages, etc.
```

#### `proto/testdata/orders.proto`

Same package (`testdata`) as `basic.proto` — tests package merging in the explorer tree.

```protobuf
syntax = "proto3";
package testdata;

service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
}
```

#### `proto/testdata/separate_pkg.proto`

Different package — tests multi-package tree display.

```protobuf
syntax = "proto3";
package payments;

service PaymentService {
  rpc ProcessPayment(ProcessPaymentRequest) returns (ProcessPaymentResponse);
}
```

---

### 3.2 `internal/ui/` — Shared UI Components

#### `internal/ui/theme.go`

```go
package ui

// Color palette constants (dark theme only).
const (
    ColorPrimary     = "#7C3AED"
    ColorSecondary   = "#06B6D4"
    ColorSuccess     = "#22C55E"
    ColorWarning     = "#EAB308"
    ColorError       = "#EF4444"
    ColorDimmed      = "#6B7280"
    ColorText        = "#F9FAFB"
    ColorBorder      = "#374151"
    ColorFocusBorder = "#7C3AED"
    ColorBackground  = "#111827"
)

// StatusColor returns the appropriate color for a gRPC status code.
func StatusColor(code codes.Code) lipgloss.Color
```

#### `internal/ui/styles.go`

```go
package ui

// Pre-built Lip Gloss styles used across all panes.
var (
    PaneBorder        lipgloss.Style // Unfocused pane border
    PaneFocusedBorder lipgloss.Style // Focused pane border (purple)
    TitleStyle        lipgloss.Style // Pane title text
    DimmedStyle       lipgloss.Style // Grayed text (placeholders, streaming)
    ErrorStyle        lipgloss.Style // Red text (validation errors)
    SuccessStyle      lipgloss.Style // Green text (OK status)
    WarningStyle      lipgloss.Style // Yellow text
    SelectedStyle     lipgloss.Style // Highlighted/selected item
    HelpBarStyle      lipgloss.Style // Bottom help bar
)
```

#### `internal/ui/help.go`

```go
package ui

// HelpBar renders a context-sensitive shortcut legend at the bottom.
type HelpBar struct { ... }

func NewHelpBar() HelpBar
func (h HelpBar) View(bindings []KeyBinding, warnings []string) string
```

`KeyBinding` is a simple struct: `{ Key string, Description string }`. Warnings (like "2 files failed to parse") render at the right side of the bar.

#### `internal/ui/logo.go`

```go
package ui

// Logo returns the ASCII art logo string.
func Logo() string

// EmptyState returns the logo + contextual hint message.
func EmptyState(hint string) string
```

---

### 3.3 `internal/app/messages.go` — Cross-Pane Messages

All `tea.Msg` types used for communication between panes. See [Section 2.2](#22-cross-pane-message-types) for the full type definitions.

This file defines: `MethodSelectedMsg`, `SendRequestMsg`, `ResponseReceivedMsg`, `FocusPaneMsg`, `PaneID`, `ProtoReloadMsg`, `ProtoLoadedMsg`, `ConnectionChangedMsg`, `TerminalTooSmallMsg`, `CopyGrpcurlMsg`.

---

### 3.4 `internal/proto/` — Proto Parsing Layer

#### `internal/proto/loader.go`

```go
package proto

import "github.com/bufbuild/protocompile"

// Loader wraps protocompile.Compiler for parsing .proto files.
type Loader struct {
    importPaths []string
}

// NewLoader creates a Loader with the given additional import paths.
// Well-known types are automatically available via WithStandardImports.
func NewLoader(importPaths []string) *Loader

// Load parses the given .proto file paths and returns a Registry.
// Partial loading: files that fail to parse are skipped, and their
// errors are returned in the warnings slice. Only returns a non-nil
// error if ALL files fail.
func (l *Loader) Load(paths []string) (reg *Registry, warnings []string, err error)
```

**Implementation notes:**
- Use `protocompile.Compiler` with `protocompile.WithStandardImports` for well-known types.
- Use `protocompile.SourceResolver` to resolve imports from `--import-path` directories.
- Iterate paths, compile each independently. On error, add to warnings and continue.

#### `internal/proto/registry.go`

```go
package proto

import "google.golang.org/protobuf/reflect/protoreflect"

// Registry holds all parsed proto descriptors, indexed for lookup.
type Registry struct { ... }

// PackageEntry represents a proto package with its services.
type PackageEntry struct {
    Name     protoreflect.FullName
    Services []ServiceEntry
}

// ServiceEntry represents a gRPC service with its methods.
type ServiceEntry struct {
    Name    protoreflect.FullName
    Desc    protoreflect.ServiceDescriptor
    Methods []MethodEntry
}

// MethodEntry represents a single RPC method.
type MethodEntry struct {
    Name        protoreflect.Name
    Desc        protoreflect.MethodDescriptor
    IsStreaming bool // true if client or server streaming
}

// Packages returns all packages sorted alphabetically.
func (r *Registry) Packages() []PackageEntry

// GetMethod looks up a method by full service name and method name.
func (r *Registry) GetMethod(service protoreflect.FullName, method protoreflect.Name) (MethodEntry, bool)
```

**Implementation notes:**
- Walk all file descriptors, group services by package name.
- Services in the same package from different files are merged.
- Services with no package use `(default)` as the package name.
- `IsStreaming` is true if `MethodDescriptor.IsStreamingClient()` or `IsStreamingServer()`.

#### `internal/proto/descriptor.go`

```go
package proto

import "google.golang.org/protobuf/reflect/protoreflect"

// FieldKind classifies proto fields for the request builder.
type FieldKind int

const (
    FieldKindScalar    FieldKind = iota
    FieldKindBool
    FieldKindEnum
    FieldKindMessage
    FieldKindRepeated
    FieldKindMap
    FieldKindOneof
    FieldKindTimestamp
    FieldKindDuration
    FieldKindStruct
    FieldKindWrapper
)

// FieldInfo describes a single field for form generation.
type FieldInfo struct {
    Path       string                        // Dot-notation path (e.g., "address.street")
    Name       string                        // Field name
    Kind       FieldKind                     // Classification
    Desc       protoreflect.FieldDescriptor  // Original descriptor
    EnumValues []protoreflect.EnumValueDescriptor // Non-nil for FieldKindEnum
    Children   []FieldInfo                   // Non-nil for FieldKindMessage, FieldKindMap
    OneofPeers []FieldInfo                   // Non-nil for FieldKindOneof
}

// ClassifyField determines the FieldKind for a given field descriptor.
func ClassifyField(fd protoreflect.FieldDescriptor) FieldKind

// WalkMessage recursively walks a message descriptor and returns
// a flat list of FieldInfo with dot-notation paths.
func WalkMessage(md protoreflect.MessageDescriptor) []FieldInfo

// IsWellKnownType returns true if the message is a recognized well-known type.
func IsWellKnownType(md protoreflect.MessageDescriptor) bool
```

#### `internal/proto/proto_test.go`

Tests using `proto/testdata/` files:
- `TestLoad_BasicProto` — parse `basic.proto`, verify services/methods extracted
- `TestLoad_PackageMerge` — parse `basic.proto` + `orders.proto`, verify same-package merge
- `TestLoad_MultiPackage` — parse all three, verify separate package nodes
- `TestLoad_PartialFailure` — parse with a bad file, verify partial load + warnings
- `TestClassifyField` — verify FieldKind for every field type
- `TestWalkMessage` — verify recursive field info extraction with correct paths
- `TestIsWellKnownType` — verify detection of Timestamp, Duration, Struct, wrappers

---

### 3.5 `internal/explorer/` — Explorer Pane

#### `internal/explorer/tree.go`

```go
package explorer

// NodeKind identifies the type of tree node.
type NodeKind int

const (
    NodePackage NodeKind = iota
    NodeService
    NodeMethod
)

// TreeNode represents a single node in the explorer tree.
type TreeNode struct {
    Kind        NodeKind
    Label       string                        // Display label
    FullName    string                        // Fully qualified name
    Depth       int                           // Indent level (0=package, 1=service, 2=method)
    Expanded    bool                          // For package/service nodes
    IsStreaming bool                          // For method nodes
    MethodDesc  protoreflect.MethodDescriptor // Non-nil for method nodes
    Children    []*TreeNode
}

// BuildTree creates the tree from a proto.Registry.
func BuildTree(reg *proto.Registry) []*TreeNode

// FlattenVisible returns only the visible nodes (respecting expand/collapse state)
// as a flat slice for rendering and cursor navigation.
func FlattenVisible(roots []*TreeNode) []*TreeNode
```

#### `internal/explorer/model.go`

```go
package explorer

// Model is the Bubble Tea model for the explorer pane.
type Model struct {
    roots   []*TreeNode
    visible []*TreeNode // flattened visible nodes
    cursor  int
    focused bool
    width   int
    height  int
}

func New(reg *proto.Registry) Model

// SetRegistry replaces the tree (used on proto reload).
func (m *Model) SetRegistry(reg *proto.Registry)

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool)

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int)

// implements tea.Model: Init, Update, View
```

**Key behaviors:**
- `j`/`Down` — move cursor down
- `k`/`Up` — move cursor up
- `Enter`/`l`/`Right` — on package/service: toggle expand; on method: emit `MethodSelectedMsg`
- `h`/`Left` — collapse current node or move to parent
- Streaming methods: selectable but emit `MethodSelectedMsg` with `IsStreaming: true`
- Visual: selected row highlighted, streaming methods dimmed with `[stream]` tag

#### `internal/explorer/view.go`

```go
package explorer

// View renders the explorer tree.
// Each node line:
//   Package:  "📦 package.name"  (or ▾/▸ prefix)
//   Service:  "  ⚙ ServiceName"
//   Method:   "    ▸ MethodName"      (normal)
//             "    ▸ MethodName [stream]"  (streaming, dimmed)
func (m Model) View() string
```

#### `internal/explorer/explorer_test.go`

- `TestBuildTree` — verify tree structure from registry
- `TestFlattenVisible` — verify expand/collapse affects visible list
- `TestNavigation` — simulate j/k/Enter keys, verify cursor and selections
- `TestStreamingMethodDisplay` — verify streaming methods are visually distinct
- `TestMethodSelection` — verify `MethodSelectedMsg` emitted on Enter

---

### 3.6 `internal/requestbuilder/` — Request Builder Pane

This is the most complex package. 8 files.

#### `internal/requestbuilder/skeleton.go`

```go
package requestbuilder

// Generate creates a list of FormField entries from a method's input
// message descriptor. Each field is classified by FieldKind and gets
// the appropriate widget configuration.
func Generate(md protoreflect.MessageDescriptor) []FormField

// FormField represents a single field in the request form.
type FormField struct {
    Info         proto.FieldInfo
    Widget       FieldWidget     // The input widget (interface)
    Value        any             // Current value (nil = not set)
    Expanded     bool            // For collapsible sections
    ValidationErr string         // Current validation error (empty = valid)
    Children     []FormField     // For message/repeated/map fields
    OneofGroup   string          // Oneof group name (empty if not in oneof)
    OneofActive  bool            // Whether this variant is selected
}
```

#### `internal/requestbuilder/fields.go`

```go
package requestbuilder

// FieldWidget is the interface for all input widgets.
type FieldWidget interface {
    // View renders the widget.
    View(focused bool) string
    // Update handles key events when focused.
    Update(msg tea.Msg) (FieldWidget, tea.Cmd)
    // Value returns the current input value as a string.
    Value() string
    // SetValue sets the value programmatically.
    SetValue(s string)
    // Placeholder returns the placeholder text.
    Placeholder() string
    // Validate checks the current value. Returns error message or "".
    Validate() string
}

// Concrete widget implementations:

type ScalarWidget struct { ... }    // text/numeric input via bubbles/textinput
type BoolWidget struct { ... }      // toggle: true/false
type EnumWidget struct { ... }      // displays current value, opens popup on Enter
type TimestampWidget struct { ... } // text input expecting RFC3339
type DurationWidget struct { ... }  // text input expecting Go duration format
type StructWidget struct { ... }    // multi-line text input for JSON
type WrapperWidget struct { ... }   // nullable scalar (text input + null toggle)
```

**Widget details:**
- `ScalarWidget`: wraps `bubbles/textinput`. Placeholder shows type (e.g., `int32, default: 0`). For numeric types, `Validate()` checks parsability and range.
- `BoolWidget`: displays `[true]` or `[false]`, toggles on Enter/Space.
- `EnumWidget`: displays current enum value name. On Enter, the model opens a popup overlay (managed by `model.go`).
- `TimestampWidget`: text input. Placeholder: `2006-01-02T15:04:05Z`. Validates RFC3339.
- `DurationWidget`: text input. Placeholder: `e.g., 5s, 1m30s`. Validates `time.ParseDuration`.
- `StructWidget`: multi-line text input for raw JSON. Validates JSON parse.
- `WrapperWidget`: text input with a null toggle. When null, the wrapper message is not set.

#### `internal/requestbuilder/validation.go`

```go
package requestbuilder

// ValidateField performs live validation on a single field.
// Returns an error message or empty string.
func ValidateField(field *FormField) string

// ValidateAll validates all fields in the form.
// Returns a list of (path, error) pairs for invalid fields.
func ValidateAll(fields []FormField) []ValidationError

type ValidationError struct {
    Path    string
    Message string
}
```

**Validation rules:**
- Scalar: type-check (int32, int64, float, double ranges), non-empty if proto2 required
- Bool: always valid
- Enum: value must be a known enum value name
- Timestamp: `time.Parse(time.RFC3339, v)`
- Duration: `time.ParseDuration(v)`
- Struct: `json.Valid([]byte(v))`
- Wrapper: valid if null, otherwise validate inner scalar
- Repeated/Map: validate each entry recursively

#### `internal/requestbuilder/metadata.go`

```go
package requestbuilder

// MetadataSection manages the collapsible metadata key-value editor.
type MetadataSection struct {
    Entries   []MetadataEntry
    Expanded  bool
    FocusIdx  int
    FocusCol  int // 0=key, 1=value
}

type MetadataEntry struct {
    Key   textinput.Model
    Value textinput.Model
}

func NewMetadataSection() MetadataSection

// AddEntry appends a new empty key-value row.
func (m *MetadataSection) AddEntry()

// RemoveEntry removes the entry at the given index.
func (m *MetadataSection) RemoveEntry(idx int)

// ToMap returns the metadata as map[string]string for gRPC headers.
func (m *MetadataSection) ToMap() map[string]string

// Update handles key events.
func (m *MetadataSection) Update(msg tea.Msg) tea.Cmd

// View renders the metadata section.
func (m *MetadataSection) View(focused bool) string
```

#### `internal/requestbuilder/grpcurl.go`

```go
package requestbuilder

// BuildGrpcurlCommand constructs a grpcurl command string from the
// current form state, metadata, and connection settings.
func BuildGrpcurlCommand(
    host string,
    plaintext bool,
    serviceName string,
    methodName string,
    fields []FormField,
    metadata map[string]string,
    tlsConfig TLSConfig,
) string
```

**Output format:**
```bash
grpcurl -plaintext \
  -d '{"name":"world","age":30}' \
  -H 'authorization: Bearer token' \
  -H 'x-request-id: abc123' \
  localhost:50051 package.Service/Method
```

For TLS: includes `-cacert`, `-cert`, `-key`, `-servername` flags as appropriate.

#### `internal/requestbuilder/model.go`

```go
package requestbuilder

// Model is the Bubble Tea model for the request builder pane.
type Model struct {
    method        protoreflect.MethodDescriptor
    metadata      MetadataSection
    fields        []FormField
    focusIdx      int        // index into flattened visible fields
    inMetadata    bool       // true if focus is in metadata section
    enumPopup     *EnumPopup // non-nil when enum overlay is open
    focused       bool
    width, height int
    viewport      viewport.Model
}

// EnumPopup is the overlay for selecting enum values.
type EnumPopup struct {
    Values   []string
    Cursor   int
    FieldIdx int // which field this popup is for
}

func New() Model

// SetMethod configures the builder for a new RPC method.
// Generates the form skeleton and resets all state.
func (m *Model) SetMethod(md protoreflect.MethodDescriptor)

// CollectValues gathers all field values as map[string]any for the codec.
func (m *Model) CollectValues() map[string]any

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool)

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int)

// implements tea.Model: Init, Update, View
```

**Key behaviors in Update:**
- When `enumPopup != nil`, all keys go to the popup (j/k/Enter/Escape)
- `j`/`Down`/`Tab` — next field (skips hidden oneof variants)
- `k`/`Up`/`Shift+Tab` — previous field
- `Enter` on collapsible section — toggle expand
- `Enter` on enum field — open popup
- `Enter` on bool field — toggle
- `Enter` on oneof selector — cycle/select variant
- Text input on scalar/timestamp/duration/struct fields — delegates to widget
- `a` on repeated/map section — add entry
- `d` on repeated/map entry — remove entry
- Live validation on every keystroke (calls `ValidateField`)

#### `internal/requestbuilder/view.go`

```go
package requestbuilder

// View renders the request builder pane.
// Layout:
//   ┌ Metadata ─────────────────────┐
//   │ key: [____]  value: [____]    │
//   │ + Add                         │
//   └───────────────────────────────┘
//   ┌ Fields ───────────────────────┐
//   │ name: [________]              │
//   │ ▾ address                     │
//   │   street: [________]          │
//   │   city: [________]            │
//   │ tags (0): [▸]                 │
//   │ ◉ phone: [________]           │  (oneof selected)
//   │ ○ slack_handle                │  (oneof not selected)
//   └───────────────────────────────┘
//
// If enum popup is open, render it as an overlay on top.
func (m Model) View() string
```

**Rendering rules:**
- Metadata section at top, always visible (collapsed/expanded)
- Fields rendered with indentation matching depth
- Oneof: `◉` for active variant, `○` for inactive
- Repeated: `▾ items (3)` when expanded, `▸ items (3)` when collapsed
- Map: similar to repeated but shows key-value pairs
- Validation errors: red text below the invalid field
- Placeholders: dimmed text in empty inputs
- Enum popup: bordered overlay centered on the enum field

#### `internal/requestbuilder/requestbuilder_test.go`

- `TestGenerate_SimpleMessage` — verify field list for basic types
- `TestGenerate_NestedMessage` — verify dot-path generation
- `TestGenerate_OneofFields` — verify oneof grouping
- `TestGenerate_RepeatedFields` — verify repeated field classification
- `TestGenerate_MapFields` — verify map field handling
- `TestGenerate_WellKnownTypes` — verify special widget assignment
- `TestValidation_NumericRange` — verify int32/int64 validation
- `TestValidation_Timestamp` — verify RFC3339 validation
- `TestValidation_Duration` — verify Go duration format
- `TestValidation_JSON` — verify Struct field JSON validation
- `TestMetadataSection` — verify add/remove/collect
- `TestBuildGrpcurlCommand` — verify command generation with various configs

---

### 3.7 `internal/grpc/` — gRPC Client Layer

#### `internal/grpc/client.go`

```go
package grpc

import "google.golang.org/grpc"

// TLSConfig holds TLS-related settings.
type TLSConfig struct {
    Plaintext  bool
    CACert     string // file path
    Cert       string // file path
    Key        string // file path
    ServerName string
}

// Client manages the persistent gRPC connection.
type Client struct {
    conn   *grpc.ClientConn
    host   string
    tls    TLSConfig
}

// NewClient creates a Client (does not connect yet).
func NewClient(host string, tls TLSConfig) *Client

// Connect establishes the gRPC connection. Returns a tea.Cmd.
func (c *Client) Connect() tea.Cmd

// Reconnect closes and re-establishes the connection.
func (c *Client) Reconnect() tea.Cmd

// ChangeHost updates the host and reconnects.
func (c *Client) ChangeHost(host string) tea.Cmd

// Conn returns the current connection (may be nil).
func (c *Client) Conn() *grpc.ClientConn

// Close cleanly shuts down the connection.
func (c *Client) Close() error
```

**Implementation notes:**
- `Connect` builds `grpc.DialOption` slice from `TLSConfig`:
  - `Plaintext` → `grpc.WithTransportCredentials(insecure.NewCredentials())`
  - Otherwise → build TLS credentials from CA cert, client cert/key, server name
- Dial timeout: 10 seconds
- Connection state monitoring not needed (manual reconnect model)

#### `internal/grpc/codec.go`

```go
package grpc

import (
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/reflect/protoreflect"
    "google.golang.org/protobuf/types/dynamicpb"
)

// BuildMessage converts form field values into a dynamicpb.Message.
// The fieldValues map uses dot-notation paths as keys.
func BuildMessage(
    md protoreflect.MessageDescriptor,
    fieldValues map[string]any,
) (*dynamicpb.Message, error)

// coerceValue converts a string input value to the appropriate
// protoreflect.Value for the given field descriptor.
func coerceValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error)
```

**coerceValue handling by type:**
- `StringKind` → pass through
- `Int32Kind`/`Sint32Kind`/`Sfixed32Kind` → `strconv.ParseInt(v, 10, 32)`
- `Int64Kind`/`Sint64Kind`/`Sfixed64Kind` → `strconv.ParseInt(v, 10, 64)`
- `Uint32Kind`/`Fixed32Kind` → `strconv.ParseUint(v, 10, 32)`
- `Uint64Kind`/`Fixed64Kind` → `strconv.ParseUint(v, 10, 64)`
- `FloatKind` → `strconv.ParseFloat(v, 32)`
- `DoubleKind` → `strconv.ParseFloat(v, 64)`
- `BoolKind` → `strconv.ParseBool(v)`
- `BytesKind` → `base64.StdEncoding.DecodeString(v)`
- `EnumKind` → lookup enum value number by name
- `MessageKind` → recurse (or handle well-known types specially)

**Well-known type codec:**
- `Timestamp`: parse RFC3339 → set `seconds` and `nanos` fields on the inner message
- `Duration`: parse Go duration → set `seconds` and `nanos`
- `Struct`: parse JSON → build nested `Struct`/`Value`/`ListValue` messages
- `Wrappers`: set the `value` field on the wrapper message

#### `internal/grpc/invoker.go`

```go
package grpc

import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/reflect/protoreflect"
    "google.golang.org/protobuf/types/dynamicpb"
)

// InvokeUnary performs a unary RPC call and returns a tea.Cmd.
// The Cmd runs the RPC in a goroutine and sends ResponseReceivedMsg.
func InvokeUnary(
    conn *grpc.ClientConn,
    method protoreflect.MethodDescriptor,
    req *dynamicpb.Message,
    md map[string]string,
) tea.Cmd
```

**Implementation notes:**
- Build full method path: `/{package}.{service}/{method}`
- Attach metadata via `metadata.NewOutgoingContext`
- Use `conn.Invoke(ctx, fullMethod, req, resp)` with `grpc.ForceCodec` for dynamic messages
- Capture response headers/trailers via `grpc.Header(&headers)`, `grpc.Trailer(&trailers)`
- Measure latency with `time.Now()` before/after
- Marshal response to JSON via `protojson.Marshal`
- On error, extract `status.FromError(err)` for rich status info
- Return `ResponseReceivedMsg` with all collected data

#### `internal/grpc/errors.go`

```go
package grpc

import "google.golang.org/grpc/status"

// ErrorDetail represents a decoded gRPC error detail.
type ErrorDetail struct {
    Type    string // e.g., "BadRequest", "DebugInfo", "RetryInfo"
    Content string // Human-readable formatted content
    Raw     []byte // Original serialized bytes (for unknown types)
}

// DecodeErrorDetails extracts and formats error details from a gRPC status.
func DecodeErrorDetails(st *status.Status) []ErrorDetail
```

**Decoded types:**
- `google.rpc.BadRequest` → list field violations (field + description)
- `google.rpc.DebugInfo` → stack trace + detail string
- `google.rpc.RetryInfo` → retry delay duration
- Unknown `Any` → type URL + base64 raw bytes

#### `internal/grpc/grpc_test.go`

- `TestMain` — start a test gRPC server using protoc-generated test service
- `TestBuildMessage_Scalars` — verify all scalar type coercions
- `TestBuildMessage_Nested` — verify nested message construction
- `TestBuildMessage_Repeated` — verify repeated field handling
- `TestBuildMessage_Map` — verify map field handling
- `TestBuildMessage_WellKnownTypes` — verify Timestamp/Duration/Struct encoding
- `TestInvokeUnary_Success` — round-trip: build → invoke → verify response
- `TestInvokeUnary_Error` — verify error status extraction
- `TestInvokeUnary_WithMetadata` — verify metadata is sent
- `TestDecodeErrorDetails` — verify error detail decoding

---

### 3.8 `internal/responseviewer/` — Response Viewer Pane

#### `internal/responseviewer/formatter.go`

```go
package responseviewer

// FormatJSON pretty-prints JSON with syntax highlighting.
// Keys, strings, numbers, booleans, and null get distinct colors.
func FormatJSON(data []byte) string

// FormatMetadata formats gRPC metadata (headers or trailers) as
// "key: value" lines with dimmed styling.
func FormatMetadata(label string, md metadata.MD) string

// FormatStatus formats the gRPC status code with color coding.
// OK=green, NOT_FOUND/INVALID_ARGUMENT=yellow, INTERNAL/UNAVAILABLE=red.
func FormatStatus(st *status.Status) string

// FormatLatency formats the duration as a human-readable string (e.g., "12ms").
func FormatLatency(d time.Duration) string

// FormatErrorDetails formats decoded error details for display.
func FormatErrorDetails(details []grpc.ErrorDetail) string
```

#### `internal/responseviewer/model.go`

```go
package responseviewer

import "charm.land/x/bubbles/v2/viewport"

// Model is the Bubble Tea model for the response viewer pane.
type Model struct {
    viewport viewport.Model
    content  string // fully formatted response content
    loading  bool   // true while RPC is in flight
    focused  bool
    width    int
    height   int
}

func New() Model

// SetResponse formats and displays a response.
func (m *Model) SetResponse(resp app.ResponseReceivedMsg)

// SetLoading shows a spinner/loading indicator.
func (m *Model) SetLoading()

// Clear resets to empty state (shows ASCII art + hints).
func (m *Model) Clear()

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool)

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int)

// implements tea.Model: Init, Update, View
```

**SetResponse formatting order:**
1. Response headers (via `FormatMetadata("Headers", resp.Headers)`)
2. Divider
3. JSON body (via `FormatJSON(resp.Body)`) — or error details if status is not OK
4. Divider
5. Response trailers (via `FormatMetadata("Trailers", resp.Trailers)`)
6. Status line: `FormatStatus(resp.Status)` + `FormatLatency(resp.Latency)`

The entire formatted string is set as the viewport content. Virtual scrolling handles arbitrarily large responses.

#### `internal/responseviewer/view.go`

```go
package responseviewer

// View renders the response viewer pane.
// When loading: show a spinner.
// When empty: show ASCII art + "Send a request to see the response" hint.
// When populated: render the viewport with formatted content.
func (m Model) View() string
```

#### `internal/responseviewer/responseviewer_test.go`

- `TestFormatJSON_PrettyPrint` — verify indentation
- `TestFormatJSON_SyntaxColors` — verify color codes for different JSON types
- `TestFormatStatus_Colors` — verify correct color per status code
- `TestFormatLatency` — verify human-readable format
- `TestFormatErrorDetails` — verify BadRequest/DebugInfo/RetryInfo formatting
- `TestFormatMetadata` — verify header/trailer formatting

---

### 3.9 `internal/app/` — Root Application

#### `internal/app/layout.go`

```go
package app

// Layout holds the computed dimensions for each pane.
type Layout struct {
    ExplorerWidth       int
    ExplorerHeight      int
    RequestBuilderWidth int
    RequestBuilderHeight int
    ResponseViewerWidth int
    ResponseViewerHeight int
    HelpBarHeight       int
}

// MinWidth is the minimum supported terminal width.
const MinWidth = 120

// MinHeight is the minimum supported terminal height.
const MinHeight = 30

// ComputeLayout calculates pane dimensions from terminal size.
// Explorer gets ~30% width. Right panes split the remaining ~70%.
// Right panes split vertically 50/50. Help bar gets 1 row.
func ComputeLayout(termWidth, termHeight int) Layout
```

#### `internal/app/keymap.go`

```go
package app

// Global keybindings intercepted before pane-specific handling.
var GlobalKeyBindings = []KeyBinding{
    {Key: "tab",        Action: "Cycle focus forward"},
    {Key: "shift+tab",  Action: "Cycle focus backward"},
    {Key: "ctrl+enter", Action: "Send request"},
    {Key: "ctrl+c",     Action: "Quit"},
    {Key: "q",          Action: "Quit (when not in text input)"},
    {Key: "?",          Action: "Toggle help overlay"},
    {Key: "ctrl+r",     Action: "Reload proto files"},
    {Key: "ctrl+y",     Action: "Copy as grpcurl"},
    {Key: "f5",         Action: "Reconnect gRPC"},
}
```

#### `internal/app/app.go`

```go
package app

// Model is the root Bubble Tea model orchestrating all panes.
type Model struct {
    explorer       explorer.Model
    requestBuilder requestbuilder.Model
    responseViewer responseviewer.Model
    grpcClient     *grpc.Client
    registry       *proto.Registry
    loader         *proto.Loader
    protoPaths     []string
    focus          PaneID
    layout         Layout
    warnings       []string // proto parse warnings
    tooSmall       bool     // terminal below minimum size
    helpVisible    bool
}

// Config holds initialization parameters from CLI flags.
type Config struct {
    ProtoPaths  []string
    Host        string
    TLS         grpc.TLSConfig
    ImportPaths []string
}

// New creates the root model.
func New(cfg Config) Model

// Init loads protos and connects to gRPC server.
func (m Model) Init() tea.Cmd

// Update routes messages to the appropriate handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)

// View renders the full application layout.
func (m Model) View() string
```

**Update routing:**
1. `tea.WindowSizeMsg` → recompute layout, check minimum size
2. `tea.KeyPressMsg` → check global bindings first, then route to focused pane
3. `MethodSelectedMsg` → if streaming, show message; otherwise forward to request builder
4. `SendRequestMsg` → build message via codec, invoke via invoker
5. `ResponseReceivedMsg` → forward to response viewer
6. `ProtoReloadMsg` → re-parse protos via loader
7. `ProtoLoadedMsg` → update registry, rebuild explorer tree
8. `ConnectionChangedMsg` → update status
9. `CopyGrpcurlMsg` → build grpcurl command, copy to clipboard

**Focus cycling:** `Tab` → Explorer → RequestBuilder → ResponseViewer → Explorer. `Shift+Tab` reverses. `q` only quits when focus is not on a text input field.

---

### 3.10 `cmd/protopilot/main.go`

```go
package main

// Entry point. Uses Cobra for CLI flag parsing.
//
// Flags:
//   --proto, -p    (required, repeatable)  Proto file paths
//   --host         (default: localhost:50051) gRPC server address
//   --plaintext    Disable TLS
//   --import-path  (repeatable) Additional proto import paths
//   --cacert       CA certificate file
//   --cert         Client certificate file
//   --key          Client private key file
//   --servername   TLS server name override
//
// Creates app.Config, instantiates app.New(cfg), starts bubbletea.NewProgram.
```

---

### 3.11 `Makefile`

```makefile
.PHONY: build run test lint clean

build:
	CGO_ENABLED=0 go build -o ./bin/protopilot ./cmd/protopilot/

run:
	go run ./cmd/protopilot/ $(ARGS)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf ./bin/
```

---

## 4. Implementation Phases

### Phase 0: Foundation

**Goal:** Establish test data, shared types, and theme infrastructure.

| File | Description |
|---|---|
| `proto/testdata/basic.proto` | UserService with all field types (scalars, enum, nested, repeated, map, oneof, well-known) |
| `proto/testdata/orders.proto` | OrderService in same package (`testdata`) — tests package merging |
| `proto/testdata/separate_pkg.proto` | PaymentService in `payments` package — tests multi-package display |
| `internal/ui/theme.go` | Dark-only color constants |
| `internal/ui/styles.go` | Lip Gloss style definitions |
| `internal/ui/help.go` | Help bar component |
| `internal/ui/logo.go` | ASCII art logo + empty state helper |
| `internal/app/messages.go` | All cross-pane Bubble Tea message types |

**Exit criteria:** Test protos compile with `protocompile`. Theme and styles render in a test harness.

### Phase 1: Proto Parsing Layer

**Goal:** Parse `.proto` files and build an in-memory registry.

| File | Description |
|---|---|
| `internal/proto/loader.go` | `protocompile.Compiler` wrapper with standard imports + custom import paths |
| `internal/proto/registry.go` | Package-grouped service/method index |
| `internal/proto/descriptor.go` | `FieldKind` enum, `FieldInfo`, `ClassifyField`, `WalkMessage` |
| `internal/proto/proto_test.go` | Tests for loading, merging, classification, walking |

**Exit criteria:** All tests pass. `Loader.Load` successfully parses all three test protos. `WalkMessage` correctly classifies all 11 field kinds.

### Phase 2: Explorer Pane

**Goal:** Navigable tree of packages, services, and methods.

| File | Description |
|---|---|
| `internal/explorer/tree.go` | `TreeNode`, `BuildTree`, `FlattenVisible` |
| `internal/explorer/model.go` | Bubble Tea model with keyboard navigation |
| `internal/explorer/view.go` | Tree rendering with icons and dimming |
| `internal/explorer/explorer_test.go` | Tree structure, navigation, selection tests |

**Exit criteria:** Explorer renders correctly from registry. j/k navigation works. Enter on method emits `MethodSelectedMsg`. Streaming methods are grayed out.

### Phase 3: Request Builder

**Goal:** Auto-generated form from proto descriptors with full field support.

| File | Description |
|---|---|
| `internal/requestbuilder/skeleton.go` | `Generate` function, `FormField` type |
| `internal/requestbuilder/fields.go` | All widget implementations (7 types) |
| `internal/requestbuilder/validation.go` | Live + submit validation |
| `internal/requestbuilder/metadata.go` | Collapsible metadata key-value section |
| `internal/requestbuilder/grpcurl.go` | `BuildGrpcurlCommand` |
| `internal/requestbuilder/model.go` | Bubble Tea model with focus, oneof, enum popup |
| `internal/requestbuilder/view.go` | Form rendering with all visual elements |
| `internal/requestbuilder/requestbuilder_test.go` | Skeleton, validation, metadata, grpcurl tests |

**Exit criteria:** Form generates for all field kinds. Oneof radio works. Repeated add/remove works. Map key-value works. Validation catches bad input. Metadata section functional. Grpcurl output is correct.

### Phase 4: gRPC Client Layer

**Goal:** Connection management, message building, and RPC invocation.

| File | Description |
|---|---|
| `internal/grpc/client.go` | Persistent connection with full TLS support |
| `internal/grpc/codec.go` | `BuildMessage` + `coerceValue` for all types |
| `internal/grpc/invoker.go` | `InvokeUnary` as `tea.Cmd` with metadata |
| `internal/grpc/errors.go` | `DecodeErrorDetails` |
| `internal/grpc/grpc_test.go` | Integration tests with test gRPC server |

**Exit criteria:** Can connect to a gRPC server. `BuildMessage` handles all types including well-known. Round-trip invocation works. Error details decode correctly.

### Phase 5: Response Viewer

**Goal:** Formatted JSON display with metadata, status, and virtual scrolling.

| File | Description |
|---|---|
| `internal/responseviewer/formatter.go` | JSON pretty-print, metadata, status, latency, error detail formatting |
| `internal/responseviewer/model.go` | Bubble Tea model with viewport |
| `internal/responseviewer/view.go` | Rendering (loading, empty, populated states) |
| `internal/responseviewer/responseviewer_test.go` | Formatter tests |

**Exit criteria:** Responses display with syntax-highlighted JSON. Headers/trailers show above/below body. Status colors are correct. Viewport scrolls smoothly.

### Phase 6: App Orchestration + CLI

**Goal:** Wire everything together into a working application.

| File | Description |
|---|---|
| `internal/app/layout.go` | `ComputeLayout` with 120x30 minimum |
| `internal/app/keymap.go` | Global keybindings |
| `internal/app/app.go` | Root model with full message routing |
| `cmd/protopilot/main.go` | Cobra CLI with all flags |
| `Makefile` | Build/test/lint targets |

**Exit criteria:** Full end-to-end: launch app → see tree → select method → fill form → send request → see response. All global keybindings work. Tab cycling works. Minimum size warning works.

### Phase 7: Polish

**Goal:** Final touches and edge case handling.

- Wire ASCII art empty states into all panes
- Enum popup overlay compositing (renders on top of form)
- Host change modal at runtime (triggered by a keybinding)
- Parse error detail viewer modal (triggered from status bar warning)
- Clipboard integration for `Ctrl+Y` (detect platform: xclip/pbcopy/clip.exe)
- Help overlay (`?` keybinding) showing full keymap

**Exit criteria:** All features from the design decisions are implemented and functional.

---

## 5. Risks & Complexity Hotspots

### HIGH Risk

**protocompile + dynamicpb compatibility**

`bufbuild/protocompile` returns `linker.Files` which implement `protoreflect.FileDescriptor` from `google.golang.org/protobuf`. `dynamicpb.NewMessage` accepts `protoreflect.MessageDescriptor` from the same module. They *should* be compatible, but this is the critical integration point — if descriptors from protocompile don't work with dynamicpb, the entire codec layer breaks. **Mitigate:** Build an end-to-end smoke test in Phase 1 that parses a proto, creates a dynamicpb message, sets fields, and marshals to JSON.

**Request Builder complexity**

The request builder handles 11 field kinds, deep nesting, oneof radio groups, repeated/map add/remove, live validation, metadata editing, scroll management, and an enum popup overlay. This is by far the largest and most complex component. **Mitigate:** Build incrementally — start with scalars only, add one field kind at a time, test each in isolation.

### MEDIUM Risk

**Bubble Tea v2 API stability**

Bubble Tea v2 uses `charm.land` vanity imports and has API changes from v1: `View()` returns a `tea.View` struct, `KeyPressMsg` is an interface, etc. Community examples are mostly v1. **Mitigate:** Build a minimal "hello world" TUI in Phase 0 to validate imports, key handling, and viewport behavior.

**Well-known type codec handling**

Converting user-friendly formats (RFC3339 timestamps, Go durations, JSON for Struct) to dynamicpb messages requires setting inner fields directly (e.g., `seconds` and `nanos` for Timestamp). This is fiddly and error-prone. **Mitigate:** Comprehensive unit tests for every well-known type in the codec layer.

**gRPC test server setup**

Integration tests need a running gRPC server. The recommended approach is to use protoc-generated code for the test server only (this doesn't violate the "no protoc for user" philosophy — it's a dev dependency). **Mitigate:** Use a simple pre-generated test service, or use dynamicpb to register services.

### LOW Risk

**Import path resolution edge cases**

Complex proto import hierarchies (circular deps, diamond imports, relative paths) can cause parsing failures. **Mitigate:** The partial-load strategy (Decision 1.17) handles this gracefully — load what we can, warn about the rest.

---

## 6. CLI Interface

### Usage

```bash
protopilot [flags]
```

### Flags

| Flag | Short | Type | Default | Required | Description |
|---|---|---|---|---|---|
| `--proto` | `-p` | `[]string` | — | Yes | Proto file paths (repeatable) |
| `--host` | — | `string` | `localhost:50051` | No | gRPC server host:port |
| `--plaintext` | — | `bool` | `false` | No | Disable TLS (use insecure connection) |
| `--import-path` | — | `[]string` | — | No | Additional proto import paths (repeatable) |
| `--cacert` | — | `string` | — | No | CA certificate file for TLS |
| `--cert` | — | `string` | — | No | Client certificate file for mTLS |
| `--key` | — | `string` | — | No | Client private key file for mTLS |
| `--servername` | — | `string` | — | No | TLS server name override (SNI) |

### Examples

```bash
# Basic usage with plaintext
protopilot --proto ./api/service.proto --host localhost:50051 --plaintext

# Multiple proto files
protopilot -p api/users.proto -p api/orders.proto --host myserver:443

# With custom import paths
protopilot -p service.proto --import-path ./vendor/proto --import-path ./third_party

# With mTLS
protopilot -p service.proto --host secure.example.com:443 \
  --cacert ca.pem --cert client.pem --key client-key.pem

# With server name override
protopilot -p service.proto --host 10.0.0.1:443 --servername api.example.com
```

---

## 7. Keybinding Reference

### Global (intercepted before pane routing)

| Key | Action |
|---|---|
| `Tab` | Cycle focus to next pane |
| `Shift+Tab` | Cycle focus to previous pane |
| `Ctrl+Enter` | Send gRPC request |
| `Ctrl+C` | Quit application |
| `q` | Quit (only when not in a text input) |
| `?` | Toggle help overlay |
| `Ctrl+R` | Reload proto files |
| `Ctrl+Y` | Copy current request as grpcurl command |
| `F5` | Reconnect gRPC connection |

### Explorer Pane

| Key | Action |
|---|---|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `Enter` / `l` / `Right` | Expand node / Select method |
| `h` / `Left` | Collapse node / Move to parent |
| `g` | Jump to top |
| `G` | Jump to bottom |

### Request Builder Pane

| Key | Action |
|---|---|
| `j` / `Down` / `Tab` | Next field |
| `k` / `Up` / `Shift+Tab` | Previous field |
| `Enter` | Toggle collapsible section / Open enum popup / Toggle bool |
| `a` | Add entry (on repeated/map section) |
| `d` | Remove entry (on repeated/map entry) |
| Typing | Edit text input (scalar, timestamp, duration, struct, metadata) |

### Request Builder — Enum Popup

| Key | Action |
|---|---|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `Enter` | Select value and close popup |
| `Escape` | Close popup without selecting |

### Request Builder — Oneof Selector

| Key | Action |
|---|---|
| `Enter` | Activate/switch to this oneof variant |

### Response Viewer Pane

| Key | Action |
|---|---|
| `j` / `Down` | Scroll down one line |
| `k` / `Up` | Scroll up one line |
| `d` / `Page Down` | Scroll down half page |
| `u` / `Page Up` | Scroll up half page |
| `g` | Scroll to top |
| `G` | Scroll to bottom |
