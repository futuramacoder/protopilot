# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Protopilot is an interactive TUI client for exploring and calling gRPC services, built in Go. It reads `.proto` files at runtime (no code generation needed), presents services/methods as a navigable tree, generates request forms, and displays formatted responses — all inside the terminal. Inspired by [openapi-tui](https://github.com/zaghaghi/openapi-tui) but for Protocol Buffers/gRPC.

## Build & Development Commands

```bash
# Build
CGO_ENABLED=0 go build -o ./bin/protopilot ./cmd/protopilot/

# Run
go run ./cmd/protopilot/ --proto ./path/to/service.proto --host localhost:50051

# Test
go test ./...                    # all tests
go test ./internal/proto/...     # single package

# Lint
golangci-lint run
```

CLI flags: `--proto` (file paths, required), `--host` (gRPC target, default `localhost:50051`), `--plaintext` (disable TLS).

## Architecture

The app uses the **Bubble Tea Elm architecture**: all state mutations happen through messages (`tea.Msg`) and `Update` functions. No shared mutable state between components.

### Three-Pane Layout

- **Left (~30% width):** Explorer — navigable tree of Package → Service → Method
- **Top-right:** Request Builder — auto-generated form from proto message descriptors
- **Bottom-right:** Response Viewer — formatted JSON with gRPC status and latency

### Key Packages

| Package | Role |
|---|---|
| `cmd/protopilot/` | Entry point, CLI arg parsing via Cobra, app bootstrap |
| `internal/app/` | Root `tea.Model`, pane orchestration, layout, global keybindings |
| `internal/explorer/` | Left pane: tree navigation, method selection |
| `internal/requestbuilder/` | Top-right pane: form generation from proto descriptors, field editing |
| `internal/responseviewer/` | Bottom-right pane: JSON formatting, status colors, latency display |
| `internal/proto/` | Runtime `.proto` file parsing via `jhump/protoreflect`, descriptor registry |
| `internal/grpc/` | Connection management, dynamic message building (codec), unary RPC invocation |
| `internal/ui/` | Shared theme, Lip Gloss styles, help bar |

### Data Flow

1. CLI args → `proto.Loader` → `proto.Registry` (parsed descriptors)
2. Explorer builds tree from Registry
3. User selects method → `MethodSelectedMsg` → RequestBuilder generates form via `skeleton.Generate()`
4. User fills fields, presses Ctrl+Enter → `SendRequestMsg` → `grpc.Codec` builds `dynamic.Message` → `grpc.Invoker` calls RPC async
5. `ResponseReceivedMsg` → ResponseViewer formats and displays result

### Focus Model

Exactly one pane is focused at a time (Tab cycles focus). Only the focused pane receives key events; global shortcuts (Tab, Ctrl+Enter, Ctrl+C, q, ?) are intercepted first.

## Coding Conventions

- Each pane package exposes `New() Model` and implements `tea.Model`
- Error handling: return errors up the stack; never `log.Fatal` in library code (only `main.go` may exit)
- No global state — all dependencies passed via constructors or model fields
- Proto field paths use dot notation: `user.address.street`
- gRPC calls run as Bubble Tea `Cmd`s (goroutines) to keep UI responsive
- Proto types map to input widgets: string→text, int→numeric with validation, bool→toggle, enum→cycle, repeated→multi-entry, message→nested fields

## Key Dependencies

- `charmbracelet/bubbletea` v2 — TUI framework (Elm architecture)
- `charmbracelet/lipgloss` v2 — terminal styling
- `charmbracelet/bubbles` v2 — reusable TUI components
- `jhump/protoreflect` — runtime proto file parsing (no `protoc` required)
- `spf13/cobra` — CLI flag parsing
- `google.golang.org/grpc` — gRPC client
- `stretchr/testify` — test assertions

## Testing

- Unit tests for proto parsing use sample files in `proto/testdata/`
- Integration tests for gRPC invoker use a test server started in `TestMain`
- UI pane tests use `tea.Test` for simulating key sequences
- Skeleton generation tests assert correct field lists from message descriptors
