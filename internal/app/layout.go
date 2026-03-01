package app

// MinWidth is the minimum supported terminal width.
const MinWidth = 120

// MinHeight is the minimum supported terminal height.
const MinHeight = 30

// Layout holds the computed dimensions for each pane.
type Layout struct {
	ExplorerWidth        int
	ExplorerHeight       int
	RequestBuilderWidth  int
	RequestBuilderHeight int
	ResponseViewerWidth  int
	ResponseViewerHeight int
	HelpBarHeight        int
}

// ComputeLayout calculates pane dimensions from terminal size.
// Explorer gets ~30% width. Right panes split the remaining ~70%.
// Right panes split vertically 50/50. Help bar gets 1 row.
func ComputeLayout(termWidth, termHeight int) Layout {
	helpHeight := 1
	usableHeight := termHeight - helpHeight

	explorerWidth := termWidth * 30 / 100
	rightWidth := termWidth - explorerWidth

	rightTopHeight := usableHeight / 2
	rightBottomHeight := usableHeight - rightTopHeight

	return Layout{
		ExplorerWidth:        explorerWidth,
		ExplorerHeight:       usableHeight,
		RequestBuilderWidth:  rightWidth,
		RequestBuilderHeight: rightTopHeight,
		ResponseViewerWidth:  rightWidth,
		ResponseViewerHeight: rightBottomHeight,
		HelpBarHeight:        helpHeight,
	}
}
