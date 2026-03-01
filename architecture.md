# Protopilot — Architecture & Claude Code Guide

## Overview

**Protopilot** is an interactive TUI client for exploring and calling gRPC services, built in Go. It reads `.proto` files, presents services/methods as a navigable tree, generates request forms, and displays formatted responses — all inside the terminal.

**Prototype** i get this idea from this rust project https://github.com/zaghaghi/openapi-tui i want to build the same thing but for proto 

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.26+ |
| TUI Framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-architecture TUI) |
| TUI Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Proto Parsing | [jhump/protoreflect](https://github.com/jhump/protoreflect) |
| gRPC Client | `google.golang.org/grpc` |
| CLI Flags | [cobra](https://github.com/spf13/cobra) |
| Testing | `testing` stdlib + [testify](https://github.com/stretchr/testify) |
| Build | `go build` with `CGO_ENABLED=0` for static binaries |

---

## Project Structure

```
protopilot/
├── cmd/
│   └── protopilot/
│       └── main.go                 # Entry point: CLI arg parsing, app bootstrap
│
├── internal/
│   ├── app/
│   │   ├── app.go                  # Root Bubble Tea model, orchestrates panes
│   │   ├── keymap.go               # Global keybinding definitions
│   │   └── layout.go               # Terminal resize handling, pane layout math
│   │
│   ├── explorer/
│   │   ├── model.go                # Left pane: tree navigation model
│   │   ├── view.go                 # Left pane: rendering
│   │   ├── tree.go                 # Tree data structure (Package→Service→Method)
│   │   └── explorer_test.go
│   │
│   ├── requestbuilder/
│   │   ├── model.go                # Top-right pane: form model
│   │   ├── view.go                 # Top-right pane: rendering
│   │   ├── fields.go               # Field type definitions and input widgets
│   │   ├── skeleton.go             # Generate blank request from proto descriptor
│   │   └── requestbuilder_test.go
│   │
│   ├── responseviewer/
│   │   ├── model.go                # Bottom-right pane: response display model
│   │   ├── view.go                 # Bottom-right pane: rendering
│   │   ├── formatter.go            # JSON pretty-print, syntax coloring
│   │   └── responseviewer_test.go
│   │
│   ├── proto/
│   │   ├── loader.go               # Load and parse .proto files via protoreflect
│   │   ├── descriptor.go           # Extract packages, services, methods, fields
│   │   ├── registry.go             # In-memory registry of loaded descriptors
│   │   └── proto_test.go
│   │
│   ├── grpc/
│   │   ├── client.go               # Dial gRPC target, manage connection lifecycle
│   │   ├── invoker.go              # Dynamic unary RPC invocation using proto descriptors
│   │   ├── codec.go                # Build protobuf messages from form field values
│   │   └── grpc_test.go
│   │
│   └── ui/
│       ├── theme.go                # Color palette, status colors (OK=green, error=red)
│       ├── styles.go               # Lip Gloss style definitions (borders, highlights)
│       └── help.go                 # Help bar / shortcut legend component
│
├── proto/
│   └── testdata/                   # Sample .proto files for development and testing
│       ├── helloworld.proto
│       └── complex_service.proto
│
├── go.mod
├── go.sum
├── Makefile                        # build, test, lint, run targets
├── README.md
├── ARCHITECTURE.md                 # This file
└── .goreleaser.yml                 # Cross-platform release config
```

---

## Module Responsibilities

### `cmd/protopilot/main.go`
- Parse CLI flags via Cobra: `--proto` (file paths, required), `--host` (gRPC target, default `localhost:50051`), `--plaintext` (disable TLS).
- Initialize the proto registry by loading specified `.proto` files.
- Create the root `app.Model` and start the Bubble Tea program.

### `internal/proto/` — Proto Parsing Layer
- **`loader.go`**: Accept file paths, parse `.proto` files using `protoreflect/desc/protoparse`. Return file descriptors.
- **`descriptor.go`**: Walk file descriptors to extract a structured representation: packages → services → methods, including input/output message descriptors with full field metadata (name, type, nested messages, enums, repeated, map).
- **`registry.go`**: Hold all parsed descriptors in memory. Provide lookup: `GetMethod(fullName) → MethodDescriptor`.

### `internal/explorer/` — Left Pane (API Navigator)
- **Tree model**: Flat list with indent levels representing Package → Service → Method hierarchy.
- **Navigation**: Up/Down arrows and `j/k` Vim keys. Enter to select a method and populate the request builder.
- **View**: Render tree with icons/prefixes (`📦`, `⚙`, `▸`) and highlight the active item.
- **State**: `cursor int`, `expanded map[string]bool`, `selectedMethod *MethodDescriptor`.

### `internal/requestbuilder/` — Top-Right Pane (Request Form)
- **`skeleton.go`**: Given a method's input message descriptor, recursively generate a list of `Field` structs representing every field (including nested messages flattened with dot-path keys like `address.city`).
- **`fields.go`**: Define field types mapping proto types → input behavior:
  - `string` → text input
  - `int32/int64/float/double` → numeric input with validation
  - `bool` → toggle
  - `enum` → dropdown/cycle selection
  - `repeated` → allow adding multiple entries
  - `message` → nested group of fields
- **Navigation**: Tab / Shift+Tab between fields, type to edit.
- **State**: `fields []Field`, `focusIndex int`, values map.

### `internal/grpc/` — gRPC Client Layer
- **`client.go`**: Manage `grpc.ClientConn`. Support plaintext and TLS. Handle dial timeout.
- **`codec.go`**: Convert the form's field values (map of dot-paths → strings) into a `dynamic.Message` (protoreflect dynamic message) by walking the message descriptor and setting fields with proper type coercion.
- **`invoker.go`**: Perform unary RPC call using `grpc.Invoke` with the dynamic message. Return the response message, gRPC status, and latency duration. All invocations run in a goroutine, sending results back via a Bubble Tea `Cmd`.

### `internal/responseviewer/` — Bottom-Right Pane (Response Display)
- Receive invocation result (response message or error).
- **`formatter.go`**: Marshal response to JSON, pretty-print with indentation, apply syntax highlighting (keys, strings, numbers, booleans in different colors).
- Display gRPC status code with color coding:
  - `OK` → green
  - `NOT_FOUND`, `INVALID_ARGUMENT` → yellow
  - `INTERNAL`, `UNAVAILABLE` → red
- Show latency in milliseconds.
- Scrollable output for large responses.

### `internal/app/` — Root Application Model
- Implements `tea.Model` with `Init`, `Update`, `View`.
- Holds child models: `explorer.Model`, `requestbuilder.Model`, `responseviewer.Model`.
- **`layout.go`**: Calculate pane dimensions based on terminal size. Left pane gets ~30% width, right panes split vertically 50/50.
- **`keymap.go`**: Global shortcuts:
  - `Tab` — cycle focus between panes
  - `Ctrl+Enter` — send request
  - `Ctrl+C` / `q` — quit
  - `?` — toggle help overlay
- Route key events to the focused pane; intercept global shortcuts first.

### `internal/ui/` — Shared Styling
- **`theme.go`**: Central color definitions. Support for light/dark terminal detection if feasible, otherwise dark-first.
- **`styles.go`**: Reusable Lip Gloss styles for borders, focused/unfocused pane borders, highlighted text, dimmed text.
- **`help.go`**: Bottom bar showing context-sensitive shortcuts.

---

## Data Flow

```
1. Startup
   CLI args → proto.Loader → proto.Registry (parsed descriptors)
                                  ↓
2. Explorer populates tree from Registry

3. User selects method in Explorer
   Explorer --MethodSelected msg--> App --SetMethod--> RequestBuilder
   RequestBuilder ← skeleton.Generate(method.InputType)

4. User fills fields, presses Ctrl+Enter
   RequestBuilder --SendRequest msg--> App
   App → grpc.Codec.Build(fields, descriptor) → dynamic.Message
   App → grpc.Invoker.Invoke(conn, method, message) → tea.Cmd (async)

5. Response arrives
   Invoker --ResponseReceived msg--> App --SetResponse--> ResponseViewer
   ResponseViewer ← formatter.Format(response, status, latency)
```

---

## Key Design Decisions

1. **Bubble Tea Elm Architecture**: All state mutations happen through messages (`tea.Msg`) and the `Update` function. No shared mutable state between panes. This makes the app predictable and testable.

2. **protoreflect Dynamic Messages**: Avoid code generation. Parse `.proto` files at runtime and build messages dynamically. This is the core enabler — users don't need `protoc` or generated Go code.

3. **Single Binary**: Build with `CGO_ENABLED=0`. Embed nothing external. The binary is the entire application.

4. **Pane Focus Model**: Exactly one pane is "focused" at a time. Only the focused pane receives key events (except global shortcuts). Visual border color indicates focus.

5. **Async gRPC Calls**: Network calls run as Bubble Tea `Cmd`s (goroutines). The UI remains responsive during RPCs. A spinner shows in the response pane while waiting.

---

## Message Types (Bubble Tea)

```go
// Key application messages passed between components
type MethodSelectedMsg struct {
    Method *desc.MethodDescriptor
}

type SendRequestMsg struct {
    Fields map[string]interface{}
    Method *desc.MethodDescriptor
}

type ResponseReceivedMsg struct {
    Body    []byte          // JSON-marshaled response
    Status  *status.Status  // gRPC status
    Latency time.Duration
    Err     error
}

type FocusPaneMsg struct {
    Pane PaneID // Explorer | RequestBuilder | ResponseViewer
}
```

---

## Build & Run

```bash
# Development
make run PROTO=./path/to/service.proto HOST=localhost:50051

# Build release binary
make build          # → ./bin/protopilot

# Cross-compile
make release        # Uses goreleaser

# Test
make test           # go test ./...

# Lint
make lint           # golangci-lint run
```

---

## Testing Strategy

| Layer | Approach |
|---|---|
| `internal/proto/` | Unit tests with sample `.proto` files in `proto/testdata/`. Verify correct tree extraction, field types, nested messages. |
| `internal/requestbuilder/` | Unit tests for skeleton generation. Given a message descriptor, assert correct field list with types and paths. |
| `internal/grpc/codec` | Unit tests for field value → dynamic message conversion. Cover type coercion edge cases (string→int, enum names). |
| `internal/grpc/invoker` | Integration test with a test gRPC server (started in `TestMain`). Verify round-trip: build message → invoke → parse response. |
| `internal/explorer/` | Unit tests for tree construction, cursor movement, expand/collapse logic. |
| `internal/responseviewer/` | Unit tests for JSON formatting, status color selection. |
| `internal/app/` | Integration tests: simulate key sequences using `tea.Test`, verify pane transitions and message routing. |

---

## Implementation Order

1. **Phase 1 — Proto Parsing**: `internal/proto/` — load files, extract tree structure. Write tests with sample protos.
2. **Phase 2 — Explorer Pane**: `internal/explorer/` + `internal/ui/` — render navigable tree. Runnable TUI showing just the tree.
3. **Phase 3 — Request Builder**: `internal/requestbuilder/` — skeleton generation, field editing. Wire to explorer selection.
4. **Phase 4 — gRPC Client**: `internal/grpc/` — connection, codec, invocation. Test with a real gRPC server.
5. **Phase 5 — Response Viewer**: `internal/responseviewer/` — formatted output, status colors, latency.
6. **Phase 6 — Integration**: `internal/app/` + `cmd/protopilot/` — wire all panes, global shortcuts, layout, polish.

---

## Coding Conventions

- Use `internal/` to prevent external imports of application packages.
- Each pane package exposes: `New() Model`, and implements `tea.Model` interface.
- Error handling: return errors up; never `log.Fatal` in library code. Only `main.go` may exit.
- Naming: follow Go conventions. Exported types for cross-package Bubble Tea messages. Unexported for internal state.
- Comments: document all exported types and functions. Describe *why*, not *what*.
- No global state. All dependencies passed via constructors or model fields.
- Proto field paths use dot notation: `user.address.street`.

---

## Dependencies (go.mod)

```
module github.com/futuramacoder/protopilot

go 1.26

require (
    github.com/charmbracelet/bubbletea   v2.0.0
    github.com/charmbracelet/lipgloss    v2.0.0
    github.com/charmbracelet/bubbles     v2.0.0
    github.com/jhump/protoreflect        v1.18.0
    github.com/spf13/cobra               v1.10.2
    google.golang.org/grpc               v1.79.1
    google.golang.org/protobuf           v1.36.11
    github.com/stretchr/testify          v1.11.1
)
```

---

## Future Considerations (Post-MVP)

These are documented for architectural awareness — do **not** implement them in MVP:

- **Server Reflection**: Add `internal/grpc/reflection.go` — query service descriptors from a running server. Feed into the same `proto.Registry`.
- **Metadata/Headers**: Add a metadata pane or modal. Store as `map[string]string`, attach via `grpc.Header` call option.
- **Streaming**: Extend `invoker.go` with `InvokeServerStream`, `InvokeClientStream`, `InvokeBidiStream`. Response viewer becomes a scrolling log.
- **Request History**: Add `internal/history/` with a file-based store (`~/.protopilot/history.json`). Show recent requests in a modal.

