package responseviewer

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/futuramacoder/protopilot/internal/messages"
	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
	"github.com/futuramacoder/protopilot/internal/ui"
)

// Model is the Bubble Tea model for the response viewer pane.
type Model struct {
	viewport viewport.Model
	content  string // fully formatted response content
	loading  bool
	hasResp  bool // true when a response has been set
	focused  bool
	width    int
	height   int
}

// New creates a new empty response viewer model.
func New() Model {
	vp := viewport.New()
	return Model{
		viewport: vp,
	}
}

// SetResponse formats and displays a response.
func (m *Model) SetResponse(resp messages.ResponseReceivedMsg) {
	m.loading = false
	m.hasResp = true

	var sections []string

	// 1. Response headers.
	if hdr := FormatMetadata("Headers", resp.Headers); hdr != "" {
		sections = append(sections, hdr)
	}

	// 2. Divider.
	dividerWidth := m.width - 4
	if dividerWidth < 1 {
		dividerWidth = 40
	}
	divider := ui.DimmedStyle.Render(strings.Repeat("─", dividerWidth))

	if len(sections) > 0 {
		sections = append(sections, divider)
	}

	// 3. Body or error.
	if resp.Err != nil {
		// Error response: show status + error + details.
		errStyle := ui.ErrorStyle
		sections = append(sections, errStyle.Render("Error: "+resp.Err.Error()))

		if resp.Status != nil {
			details := grpcpkg.DecodeErrorDetails(resp.Status)
			if detailStr := FormatErrorDetails(details); detailStr != "" {
				sections = append(sections, "")
				sections = append(sections, detailStr)
			}
		}
	} else if len(resp.Body) > 0 {
		sections = append(sections, FormatJSON(resp.Body))
	} else {
		sections = append(sections, ui.DimmedStyle.Render("(empty response body)"))
	}

	// 4. Divider + trailers.
	if trl := FormatMetadata("Trailers", resp.Trailers); trl != "" {
		sections = append(sections, divider)
		sections = append(sections, trl)
	}

	// 5. Status + latency footer.
	sections = append(sections, divider)
	var footer []string
	if resp.Status != nil {
		footer = append(footer, FormatStatus(resp.Status))
	}
	footer = append(footer, FormatLatency(resp.Latency))
	sections = append(sections, strings.Join(footer, "  "))

	m.content = strings.Join(sections, "\n")
	m.viewport.SetContent(m.content)
	m.viewport.GotoTop()
}

// SetLoading shows a loading indicator.
func (m *Model) SetLoading() {
	m.loading = true
	m.hasResp = false
	m.content = ""
}

// Clear resets to empty state.
func (m *Model) Clear() {
	m.loading = false
	m.hasResp = false
	m.content = ""
}

// SetFocused sets whether this pane has keyboard focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the pane dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Account for border (2) and title line (1).
	innerHeight := height - 3
	innerWidth := width - 2
	if innerHeight < 1 {
		innerHeight = 1
	}
	if innerWidth < 1 {
		innerWidth = 1
	}
	m.viewport.SetWidth(innerWidth)
	m.viewport.SetHeight(innerHeight)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg.(type) {
	case tea.KeyPressMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}
