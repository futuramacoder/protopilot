package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/futuramacoder/protopilot/internal/explorer"
	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
	"github.com/futuramacoder/protopilot/internal/proto"
	"github.com/futuramacoder/protopilot/internal/requestbuilder"
	"github.com/futuramacoder/protopilot/internal/responseviewer"
	"github.com/futuramacoder/protopilot/internal/ui"
)

// Config holds initialization parameters from CLI flags.
type Config struct {
	ProtoPaths  []string
	Host        string
	TLS         grpcpkg.TLSConfig
	ImportPaths []string
}

// Model is the root Bubble Tea model orchestrating all panes.
type Model struct {
	explorer       explorer.Model
	requestBuilder requestbuilder.Model
	responseViewer responseviewer.Model
	grpcClient     *grpcpkg.Client
	registry       *proto.Registry
	loader         *proto.Loader
	protoPaths     []string
	focus          PaneID
	layout         Layout
	warnings       []string
	connected      bool
	tooSmall       bool
	helpVisible    bool
	warningsModal  bool
	hostModal      HostModal
	statusMsg      string // transient status (e.g., "Copied!")
	helpBar        ui.HelpBar
	termWidth      int
	termHeight     int
}

// New creates the root model.
func New(cfg Config) Model {
	loader := proto.NewLoader(cfg.ImportPaths)
	client := grpcpkg.NewClient(cfg.Host, cfg.TLS)

	m := Model{
		explorer:       explorer.New(nil),
		requestBuilder: requestbuilder.New(),
		responseViewer: responseviewer.New(),
		grpcClient:     client,
		loader:         loader,
		protoPaths:     cfg.ProtoPaths,
		focus:          PaneExplorer,
		helpBar:        ui.NewHelpBar(),
		hostModal:      NewHostModal(),
	}
	m.explorer.SetFocused(true)

	return m
}

// Init loads protos and connects to gRPC server.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadProtos(),
		m.grpcClient.Connect(),
	)
}

func (m Model) loadProtos() tea.Cmd {
	loader := m.loader
	paths := m.protoPaths
	return func() tea.Msg {
		reg, warnings, err := loader.Load(paths)
		return ProtoLoadedMsg{
			Registry: reg,
			Warnings: warnings,
			Err:      err,
		}
	}
}

// Update routes messages to the appropriate handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyPressMsg:
		if m.tooSmall {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		return m.handleKey(msg)

	case ProtoLoadedMsg:
		return m.handleProtoLoaded(msg)

	case ConnectionChangedMsg:
		return m.handleConnectionChanged(msg)

	case MethodSelectedMsg:
		return m.handleMethodSelected(msg)

	case SendRequestMsg:
		return m.handleSendRequest(msg)

	case ResponseReceivedMsg:
		return m.handleResponseReceived(msg)

	case ClipboardResultMsg:
		return m.handleClipboardResult(msg)
	}

	// Route to focused pane.
	return m.updateFocusedPane(msg)
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.termWidth = msg.Width
	m.termHeight = msg.Height

	if msg.Width < MinWidth || msg.Height < MinHeight {
		m.tooSmall = true
		return m, nil
	}

	m.tooSmall = false
	m.layout = ComputeLayout(msg.Width, msg.Height)

	m.explorer.SetSize(m.layout.ExplorerWidth, m.layout.ExplorerHeight)
	m.requestBuilder.SetSize(m.layout.RequestBuilderWidth, m.layout.RequestBuilderHeight)
	m.responseViewer.SetSize(m.layout.ResponseViewerWidth, m.layout.ResponseViewerHeight)
	m.helpBar.SetWidth(msg.Width)

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Clear transient status on any key press.
	m.statusMsg = ""

	// Host modal captures all input when visible.
	if m.hostModal.Visible() {
		confirmed, cmd := m.hostModal.Update(msg)
		if confirmed {
			newHost := m.hostModal.Value()
			if newHost != "" {
				return m, m.grpcClient.ChangeHost(newHost)
			}
		}
		return m, cmd
	}

	// Overlays: help and warnings dismiss on Esc or their toggle key.
	if m.helpVisible {
		if key == "?" || key == "escape" || key == "esc" {
			m.helpVisible = false
		}
		return m, nil
	}
	if m.warningsModal {
		if key == "ctrl+w" || key == "escape" || key == "esc" {
			m.warningsModal = false
		}
		return m, nil
	}

	switch key {
	case "ctrl+c":
		return m, tea.Quit

	case "q":
		if m.focus != PaneRequestBuilder {
			return m, tea.Quit
		}

	case "tab":
		m.cycleFocus(1)
		return m, nil

	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil

	case "ctrl+enter":
		return m.handleCtrlEnter()

	case "?":
		m.helpVisible = true
		return m, nil

	case "ctrl+r":
		return m, m.loadProtos()

	case "ctrl+y":
		return m.handleCopyGrpcurl()

	case "ctrl+h":
		cmd := m.hostModal.Show(m.grpcClient.Host())
		return m, cmd

	case "ctrl+w":
		m.warningsModal = true
		return m, nil

	case "f5":
		return m, m.grpcClient.Reconnect()
	}

	// Route to focused pane.
	return m.updateFocusedPane(msg)
}

func (m *Model) cycleFocus(delta int) {
	paneCount := 3
	next := (int(m.focus) + delta + paneCount) % paneCount
	m.focus = PaneID(next)
	m.applyFocus()
}

func (m *Model) applyFocus() {
	m.explorer.SetFocused(m.focus == PaneExplorer)
	m.requestBuilder.SetFocused(m.focus == PaneRequestBuilder)
	m.responseViewer.SetFocused(m.focus == PaneResponseViewer)
}

func (m Model) handleCtrlEnter() (tea.Model, tea.Cmd) {
	result, cmd := m.requestBuilder.Update(tea.KeyPressMsg(tea.Key{Code: -1, Text: "ctrl+enter"}))
	if rb, ok := result.(requestbuilder.Model); ok {
		m.requestBuilder = rb
	}
	return m, cmd
}

func (m Model) handleCopyGrpcurl() (tea.Model, tea.Cmd) {
	method := m.requestBuilder.Method()
	if method == nil {
		m.statusMsg = "No method selected"
		return m, nil
	}

	svc := method.Parent()
	svcName := string(svc.FullName())
	methodName := string(method.Name())

	cmd := requestbuilder.BuildGrpcurlCommand(
		m.grpcClient.Host(),
		m.grpcClient.TLS().Plaintext,
		svcName,
		methodName,
		m.requestBuilder.Fields(),
		m.requestBuilder.MetadataMap(),
		requestbuilder.TLSConfig{
			Plaintext:  m.grpcClient.TLS().Plaintext,
			CACert:     m.grpcClient.TLS().CACert,
			Cert:       m.grpcClient.TLS().Cert,
			Key:        m.grpcClient.TLS().Key,
			ServerName: m.grpcClient.TLS().ServerName,
		},
	)

	return m, copyToClipboard(cmd)
}

func (m Model) handleClipboardResult(msg ClipboardResultMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.statusMsg = "Copied grpcurl command!"
	} else {
		m.statusMsg = fmt.Sprintf("Clipboard error: %v", msg.Err)
	}
	return m, nil
}

func (m Model) handleProtoLoaded(msg ProtoLoadedMsg) (tea.Model, tea.Cmd) {
	m.warnings = msg.Warnings

	if msg.Err != nil {
		m.warnings = append(m.warnings, msg.Err.Error())
		return m, nil
	}

	m.registry = msg.Registry
	m.explorer.SetRegistry(msg.Registry)

	return m, nil
}

func (m Model) handleConnectionChanged(msg ConnectionChangedMsg) (tea.Model, tea.Cmd) {
	m.connected = msg.Connected
	if msg.Err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("connection: %v", msg.Err))
	}
	return m, nil
}

func (m Model) handleMethodSelected(msg MethodSelectedMsg) (tea.Model, tea.Cmd) {
	if msg.IsStreaming {
		m.responseViewer.Clear()
		return m, nil
	}

	m.requestBuilder.SetMethod(msg.MethodDesc)

	m.focus = PaneRequestBuilder
	m.applyFocus()

	m.responseViewer.Clear()

	return m, nil
}

func (m Model) handleSendRequest(msg SendRequestMsg) (tea.Model, tea.Cmd) {
	if m.grpcClient.Conn() == nil {
		m.statusMsg = "Not connected to gRPC server"
		return m, nil
	}

	dynMsg, err := grpcpkg.BuildMessage(msg.MethodDesc.Input(), msg.FieldValues)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Build error: %v", err)
		return m, nil
	}

	m.responseViewer.SetLoading()

	m.focus = PaneResponseViewer
	m.applyFocus()

	return m, grpcpkg.InvokeUnary(m.grpcClient.Conn(), msg.MethodDesc, dynMsg, msg.Metadata)
}

func (m Model) handleResponseReceived(msg ResponseReceivedMsg) (tea.Model, tea.Cmd) {
	m.responseViewer.SetResponse(msg)
	return m, nil
}

func (m Model) updateFocusedPane(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.focus {
	case PaneExplorer:
		var result tea.Model
		result, cmd = m.explorer.Update(msg)
		if e, ok := result.(explorer.Model); ok {
			m.explorer = e
		}
	case PaneRequestBuilder:
		var result tea.Model
		result, cmd = m.requestBuilder.Update(msg)
		if rb, ok := result.(requestbuilder.Model); ok {
			m.requestBuilder = rb
		}
	case PaneResponseViewer:
		var result tea.Model
		result, cmd = m.responseViewer.Update(msg)
		if rv, ok := result.(responseviewer.Model); ok {
			m.responseViewer = rv
		}
	}

	return m, cmd
}

// View renders the full application layout.
func (m Model) View() tea.View {
	if m.tooSmall {
		msg := lipgloss.NewStyle().
			Foreground(ui.ColorWarning).
			Bold(true).
			Render(fmt.Sprintf(
				"Terminal too small (%dx%d). Please resize to at least %dx%d.",
				m.termWidth, m.termHeight, MinWidth, MinHeight,
			))

		content := lipgloss.NewStyle().
			Width(m.termWidth).
			Height(m.termHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(msg)

		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}

	// Check for overlay modals.
	if m.helpVisible {
		v := tea.NewView(ui.HelpOverlay(m.termWidth, m.termHeight))
		v.AltScreen = true
		return v
	}

	if m.warningsModal {
		v := tea.NewView(ui.WarningsOverlay(m.warnings, m.termWidth, m.termHeight))
		v.AltScreen = true
		return v
	}

	if m.hostModal.Visible() {
		v := tea.NewView(m.hostModal.View(m.termWidth, m.termHeight))
		v.AltScreen = true
		return v
	}

	// Render each pane.
	explorerView := m.explorer.View().Content
	requestView := m.requestBuilder.View().Content
	responseView := m.responseViewer.View().Content

	// Right column: request builder on top, response viewer below.
	rightColumn := lipgloss.JoinVertical(lipgloss.Left,
		requestView,
		responseView,
	)

	// Main layout: explorer on left, right column on right.
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		explorerView,
		rightColumn,
	)

	// Help bar.
	var warningStrs []string
	if len(m.warnings) > 0 {
		warningStrs = append(warningStrs, fmt.Sprintf("%d warning(s)", len(m.warnings)))
	}
	if !m.connected {
		warningStrs = append(warningStrs, "disconnected")
	}
	if m.statusMsg != "" {
		warningStrs = append(warningStrs, m.statusMsg)
	}
	helpView := m.helpBar.View(DefaultHelpBindings(), warningStrs)

	full := lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		helpView,
	)

	v := tea.NewView(full)
	v.AltScreen = true
	return v
}
