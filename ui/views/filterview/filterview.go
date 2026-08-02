package filterview

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"skim/filterfiles"
	"strings"
)

// filterColumn identifies which toggleable column of the filter row is
// currently selected via left/right (h/l) navigation.
type filterColumn int

const (
	EnabledColumn       filterColumn = iota // the enabled checkbox
	CaseSensitiveColumn                     // the case-sensitivity checkbox
	maxFilterColumn                         // unused, represents the total number of columns
)

// visibleHeight is how many filter rows are shown at once (matches the
// previous bubbles/table WithHeight(5)).
const visibleHeight = 5

type FilterView struct {
	Cursor  int          // which filter our cursor is pointing at
	Column  filterColumn // which toggleable column is selected
	Filters []filterfiles.Filter
}

// Toggle flips whichever checkbox column is currently selected for the
// filter under the cursor.
func (v *FilterView) Toggle() {
	filter := &v.Filters[v.Cursor]
	switch v.Column {
	case EnabledColumn:
		filter.IsEnabled = !filter.IsEnabled
	case CaseSensitiveColumn:
		filter.CaseSensitive = !filter.CaseSensitive
		if regex, err := filterfiles.CompileRegex(filter.XML.Text, filter.CaseSensitive); err == nil {
			filter.Regex = regex
		}
	}
}

func (v *FilterView) CursorUp() int {
	if v.Cursor > 0 {
		v.Cursor--
	}
	return v.Cursor
}

func (v *FilterView) CursorDown() int {
	if v.Cursor < v.GetMaxCursor() {
		v.Cursor++
	}
	return v.Cursor
}

func (v *FilterView) CursorLeft() int {
	if v.Column > 0 {
		v.Column--
	}
	return int(v.Column)
}

func (v *FilterView) CursorRight() int {
	if v.Column < maxFilterColumn-1 {
		v.Column++
	}
	return int(v.Column)
}

func (v *FilterView) GetMaxCursor() int {
	return len(v.Filters) - 1
}

// UpdateRegexText replaces the regex text of the filter at index, recompiling
// its regexp. If the new text fails to compile, the filter is left unchanged
// and the compile error is returned.
func (v *FilterView) UpdateRegexText(index int, text string) error {
	if index < 0 || index >= len(v.Filters) {
		return fmt.Errorf("filter index %d out of range", index)
	}

	regex, err := filterfiles.CompileRegex(text, v.Filters[index].CaseSensitive)
	if err != nil {
		return err
	}

	v.Filters[index].XML.Text = text
	v.Filters[index].Regex = regex
	return nil
}

var filterStyle = lipgloss.NewStyle().
	Bold(false).
	Foreground(lipgloss.Color("#000000")).
	PaddingTop(0).
	PaddingLeft(0)

var headerStyle = lipgloss.NewStyle().Bold(true)

// selectedCellStyle marks exactly the cell under the row+column cursor,
// matching bubbles/table's default selected-row look (bold, pink). Rows
// are rendered by hand (see Render) rather than through bubbles/table,
// since that library width-constrains cell content with a plain rune
// count before applying any style, which mangles short ANSI-styled
// strings like this one.
var selectedCellStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

// cell pads content to an exact visible width, ANSI-aware, so already
// styled strings (e.g. from selectedCellStyle) aren't corrupted.
func cell(content string, width int) string {
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true).Render(content)
}

// Render draws the filters table: a header row plus one row per filter,
// each with an enabled checkbox, a live match count, the regex text
// (colored by the filter's BackColor), and a case-sensitivity checkbox. The
// row/column under the cursor is marked with braces instead of brackets,
// and highlighted pink via selectedCellStyle, so the highlight always
// matches the cell that enter/space would toggle.
//
// counts holds each filter's current match count, indexed the same as
// v.Filters (see filterfiles.CountMatches); a short or nil counts is
// treated as all-zero, so callers that don't have counts handy can pass nil.
func (v *FilterView) Render(windowWidth int, windowHeight int, counts []int) string {
	enabledWidth := 3
	countWidth := 6
	caseWidth := 4
	regexWidth := windowWidth - 16 - countWidth - 1 // TODO: Avoid hardcoding this offset

	var b strings.Builder

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		cell("", enabledWidth), " ",
		headerStyle.Render(cell("#", countWidth)), " ",
		headerStyle.Render(cell("Regex", regexWidth)), " ",
		headerStyle.Render(cell("Aa", caseWidth)),
	)
	b.WriteString(header)
	b.WriteString("\n")

	start := 0
	if v.Cursor >= visibleHeight {
		start = v.Cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(v.Filters) {
		end = len(v.Filters)
	}

	for i := start; i < end; i++ {
		filter := v.Filters[i]

		checked := " " // not selected
		if filter.IsEnabled {
			checked = "x" // this item is selected
		}
		// Use braces instead of brackets to mark the column selected via
		// left/right (h/l) navigation, on the row under the cursor.
		open, close := "[", "]"
		if i == v.Cursor && v.Column == EnabledColumn {
			open, close = "{", "}"
		}
		enabledCell := fmt.Sprintf("%s%s%s", open, checked, close)
		if i == v.Cursor && v.Column == EnabledColumn {
			enabledCell = selectedCellStyle.Render(enabledCell)
		}

		caseChecked := " "
		if filter.CaseSensitive {
			caseChecked = "x"
		}
		open, close = "[", "]"
		if i == v.Cursor && v.Column == CaseSensitiveColumn {
			open, close = "{", "}"
		}
		caseCell := fmt.Sprintf("%s%s%s", open, caseChecked, close)
		if i == v.Cursor && v.Column == CaseSensitiveColumn {
			caseCell = selectedCellStyle.Render(caseCell)
		}

		style := filterStyle
		style.Background(lipgloss.Color(filter.BackColor))
		regexCell := style.Render(cell(filter.XML.Text, regexWidth))

		count := 0
		if i < len(counts) {
			count = counts[i]
		}
		countCell := cell(fmt.Sprintf("%d", count), countWidth)

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			cell(enabledCell, enabledWidth), " ",
			countCell, " ",
			regexCell, " ",
			cell(caseCell, caseWidth),
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}
