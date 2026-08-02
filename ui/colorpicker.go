package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// colorPickerCols is the number of swatches per row in the palette grid.
const colorPickerCols = 8

// colorPalette is a curated set of colors reasonable as log-line highlight
// backgrounds (skim renders filter-matched text in black, see filterStyle in
// filterview.go, so lighter/pastel colors read best -- though the grid also
// includes some more saturated and grayscale options). Stored as uppercase
// "#RRGGBB" to match Filter.BackColor's own format (see filterfiles.makeFilter).
// Arranged in colorPickerCols-wide rows for the grid layout in
// renderColorPicker; colorPickerCellAt must stay in sync with any layout
// change here.
var colorPalette = []string{
	"#FFB3BA", "#FFDFBA", "#FFFFBA", "#BAFFC9", "#BAE1FF", "#C9BAFF", "#FFBAF0", "#D3D3D3",
	"#FF6961", "#FFB347", "#FDFD96", "#77DD77", "#87CEFA", "#AEC6CF", "#CFCFC4", "#B39EB5",
	"#FF0000", "#FF8C00", "#FFD700", "#00FF00", "#00BFFF", "#1E90FF", "#9370DB", "#FF69B4",
	"#FFFFFF", "#E0E0E0", "#C0C0C0", "#808080", "#404040", "#000000", "#8B4513", "#2F4F4F",
}

// colorPickerRows is how many grid rows colorPalette fills at colorPickerCols
// swatches per row (the last row may be partially filled).
func colorPickerRows() int {
	return (len(colorPalette) + colorPickerCols - 1) / colorPickerCols
}

// colorPaletteIndexFor returns the palette index matching hex (case
// insensitive), or 0 if hex isn't one of the preset swatches -- so opening
// the picker on a filter with a custom color just starts the cursor at the
// first swatch rather than erroring.
func colorPaletteIndexFor(hex string) int {
	hex = strings.ToUpper(hex)
	for i, c := range colorPalette {
		if c == hex {
			return i
		}
	}
	return 0
}

// colorPickerState holds the state of the color picker sub-screen, opened
// from the filter editor's Color field (see fieldColor in filtereditor.go).
type colorPickerState struct {
	open   bool
	cursor int // index into colorPalette

	customEditing bool   // typing a custom hex value instead of using the grid
	customBuf     string // in-progress hex digits typed so far, no leading '#'
	customErr     string // set if customBuf failed to validate on enter
}

// updateColorPicker handles key presses while the color picker is open,
// routing to custom-hex text capture if that's active.
func (m model) updateColorPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterEditor.colorPicker.customEditing {
		return m.updateColorPickerCustomInput(msg)
	}

	cp := &m.filterEditor.colorPicker

	switch msg.String() {
	case "esc":
		cp.open = false

	case "enter", " ":
		m.applyPickedColor(colorPalette[cp.cursor])
		cp.open = false

	case "c":
		cp.customEditing = true
		cp.customBuf = ""
		cp.customErr = ""

	case "up", "k":
		if cp.cursor-colorPickerCols >= 0 {
			cp.cursor -= colorPickerCols
		}

	case "down", "j":
		if cp.cursor+colorPickerCols < len(colorPalette) {
			cp.cursor += colorPickerCols
		}

	case "left", "h":
		if cp.cursor%colorPickerCols > 0 {
			cp.cursor--
		}

	case "right", "l":
		if cp.cursor%colorPickerCols < colorPickerCols-1 && cp.cursor+1 < len(colorPalette) {
			cp.cursor++
		}
	}

	return m, nil
}

// updateColorPickerCustomInput handles key presses while typing a custom hex
// color (after pressing "c" in the grid): hex digits are appended to
// customBuf (capped at 6), backspace removes the last one, esc returns to
// the grid without applying anything, and enter validates and applies it.
func (m model) updateColorPickerCustomInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cp := &m.filterEditor.colorPicker

	switch msg.String() {
	case "esc":
		cp.customEditing = false
		cp.customBuf = ""
		cp.customErr = ""

	case "enter":
		if len(cp.customBuf) != 6 {
			cp.customErr = "hex color must be 6 digits, e.g. 87CEFA"
			break
		}
		m.applyPickedColor("#" + strings.ToUpper(cp.customBuf))
		cp.customEditing = false
		cp.customBuf = ""
		cp.customErr = ""
		cp.open = false

	case "backspace":
		if len(cp.customBuf) > 0 {
			cp.customBuf = cp.customBuf[:len(cp.customBuf)-1]
		}

	default:
		for _, r := range msg.Runes {
			if isHexDigit(r) && len(cp.customBuf) < 6 {
				cp.customBuf += string(r)
			}
		}
	}

	return m, nil
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// applyPickedColor sets the currently-edited filter's BackColor. Filters are
// always non-empty here: the color picker is only reachable through the
// filter editor's Color field, which activateFilterEditorField already
// guards behind len(m.filters.Filters) > 0.
func (m *model) applyPickedColor(hex string) {
	m.filters.Filters[m.filters.Cursor].BackColor = hex
	m.filtersDirty = true
	m.saveStatus = ""
}

// updateColorPickerMouse handles mouse events while the color picker grid is
// open (not while typing a custom hex value, which has no mouse target):
// hovering (MouseMotion) moves the highlighted swatch, and clicking
// (MouseLeft) selects it and closes the picker, same as pressing enter.
func (m model) updateColorPickerMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.filterEditor.colorPicker.customEditing {
		return m, nil
	}

	idx, ok := colorPickerCellAt(msg.X, msg.Y)
	if !ok {
		return m, nil
	}

	switch msg.Type {
	case tea.MouseMotion:
		m.filterEditor.colorPicker.cursor = idx

	case tea.MouseLeft:
		m.filterEditor.colorPicker.cursor = idx
		m.applyPickedColor(colorPalette[idx])
		m.filterEditor.colorPicker.open = false
	}

	return m, nil
}

// colorPickerHeader is the picker's first line of on-screen help text, also
// used by colorPickerGridStartY to derive how many lines precede the grid,
// so the two can't silently drift out of sync.
const colorPickerHeader = "Pick a Color  —  arrows/hjkl: move   enter/click: select   c: custom hex   esc: cancel\n\n"

// colorPickerGridStartX/Y are the terminal column/row (0-indexed) of the
// first swatch's opening bracket, used by both renderColorPicker (indirectly,
// via colorPickerGridIndent) and colorPickerCellAt to translate a mouse
// click back into a grid index. Two components make up the offset:
//   - colorPickerGridIndent / the line count of colorPickerHeader: the
//     picker's own content layout.
//   - colorPickerBorderWidth: baseStyle (see ui.go), which the picker's
//     content is rendered inside of, draws a border that shifts everything
//     by one more cell down and right.
//
// If renderColorPicker's layout changes, these must change with it.
const (
	colorPickerGridIndent  = 2 // left padding before each grid row's swatches
	colorPickerBorderWidth = 1 // baseStyle's top/left border
)

func colorPickerGridStartX() int {
	return colorPickerGridIndent + colorPickerBorderWidth
}

func colorPickerGridStartY() int {
	return strings.Count(colorPickerHeader, "\n") + colorPickerBorderWidth
}

// colorPickerCellW is the visible width of one rendered swatch, "{" or "[" +
// 4 spaces of colored background + "}" or "]" (see renderColorSwatch).
const colorPickerCellW = 6

// colorPickerColStride is the horizontal distance between the start of one
// swatch and the next: its own width plus the single-space gap
// renderColorPicker puts after each one.
const colorPickerColStride = colorPickerCellW + 1

// colorPickerCellAt maps a terminal (x, y) to a colorPalette index, if it
// falls within the rendered grid (not the gap between swatches, and not past
// the last swatch on a partially-filled final row).
func colorPickerCellAt(x, y int) (int, bool) {
	row := y - colorPickerGridStartY()
	col := x - colorPickerGridStartX()
	if row < 0 || col < 0 {
		return 0, false
	}
	if row >= colorPickerRows() {
		return 0, false
	}
	if col%colorPickerColStride >= colorPickerCellW {
		return 0, false
	}
	c := col / colorPickerColStride
	if c >= colorPickerCols {
		return 0, false
	}
	idx := row*colorPickerCols + c
	if idx >= len(colorPalette) {
		return 0, false
	}
	return idx, true
}

// renderColorPicker draws either the swatch grid or, while cp.customEditing,
// the custom hex input screen.
func (m model) renderColorPicker() string {
	cp := m.filterEditor.colorPicker

	if cp.customEditing {
		return m.renderColorPickerCustomInput()
	}

	var b strings.Builder
	b.WriteString(colorPickerHeader)

	for row := 0; row < colorPickerRows(); row++ {
		b.WriteString(strings.Repeat(" ", colorPickerGridIndent))
		for col := 0; col < colorPickerCols; col++ {
			idx := row*colorPickerCols + col
			if idx >= len(colorPalette) {
				break
			}
			b.WriteString(renderColorSwatch(colorPalette[idx], idx == cp.cursor))
			b.WriteString(" ")
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("\nSelected: %s\n", colorPalette[cp.cursor]))

	return baseStyle.Render(b.String())
}

// renderColorSwatch renders a single grid cell: a 4-cell block of the
// swatch's own background color, flanked by brackets -- braces and
// selectedCellStyle's pink/bold if this is the cursor's cell, plain square
// brackets otherwise, matching the bracket/brace selection convention
// filterview.Render already uses.
// pickerSelectedStyle marks the cursor's swatch, matching the pink/bold
// convention filterview's own (unexported) selectedCellStyle uses for its
// row/column selection.
var pickerSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

func renderColorSwatch(hex string, selected bool) string {
	swatch := lipgloss.NewStyle().Background(lipgloss.Color(hex)).Render("    ")
	if selected {
		return pickerSelectedStyle.Render("{") + swatch + pickerSelectedStyle.Render("}")
	}
	return "[" + swatch + "]"
}

func (m model) renderColorPickerCustomInput() string {
	cp := m.filterEditor.colorPicker

	var b strings.Builder
	b.WriteString("Custom Hex Color  —  enter: apply   esc: back to grid\n\n")
	b.WriteString(fmt.Sprintf("  #%s\n", cp.customBuf))
	if cp.customErr != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", cp.customErr))
	}

	return baseStyle.Render(b.String())
}
