package ui

import (
	"fmt"
	"os"
	"os/exec"
	"skim/filterfiles"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// filterEditorField identifies one row of the filter editor form.
type filterEditorField int

const (
	fieldDescription filterEditorField = iota
	fieldRegex
	fieldCaseSensitive
	fieldExcluding
	fieldEnabled
	fieldColor
	maxFilterEditorField // unused, represents the total number of fields
)

// filterEditorState holds the state of the filter editor modal. It always
// edits the filter at m.filters.Cursor in place (see the EditRegex/NewFilter
// cases in Update) rather than a separate draft copy: each field commits its
// change as soon as it's confirmed (enter), mirroring the already-immediate
// checkbox toggles in filterview.Render. There is no whole-form "cancel" --
// esc just closes the modal, leaving whatever fields were already confirmed.
type filterEditorState struct {
	cursor filterEditorField

	editingText bool   // capturing text for fieldDescription or fieldRegex
	textBuf     string // in-progress text for the field being edited
	regexErr    string // set if textBuf failed to compile as a regex (fieldRegex only)

	colorPicker colorPickerState
}

// updateFilterEditor routes key presses within the filter editor modal:
// first to the color picker sub-screen if it's open, then to text-capture if
// a field is being edited, otherwise to row navigation/activation.
func (m model) updateFilterEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterEditor.colorPicker.open {
		return m.updateColorPicker(msg)
	}

	if m.filterEditor.editingText {
		return m.updateFilterEditorTextInput(msg)
	}

	switch msg.String() {
	case "esc", "q":
		m.editingFilter = false

	case "up", "k":
		if m.filterEditor.cursor > 0 {
			m.filterEditor.cursor--
		}

	case "down", "j":
		if m.filterEditor.cursor < maxFilterEditorField-1 {
			m.filterEditor.cursor++
		}

	case "enter", " ":
		return m.activateFilterEditorField()

	case "ctrl+e":
		return m.openExternalEditorForHoveredField()
	}

	return m, nil
}

// openExternalEditorForHoveredField opens ctrl+e's external-editor shortcut
// directly from a hovered Description/Regex row, without first requiring
// enter to enter text-edit mode -- a shortcut for power users over
// enter-then-ctrl+e (still available too, see updateFilterEditorTextInput,
// for switching to $EDITOR partway through an in-progress edit). It's a
// no-op on any other field, or with no filter selected.
func (m model) openExternalEditorForHoveredField() (tea.Model, tea.Cmd) {
	if len(m.filters.Filters) == 0 {
		return m, nil
	}
	filter := m.filters.Filters[m.filters.Cursor]

	var initialText string
	switch m.filterEditor.cursor {
	case fieldDescription:
		initialText = filter.XML.Description
	case fieldRegex:
		initialText = filter.XML.Text
	default:
		return m, nil
	}

	return m, openFilterFieldEditorCmd(m.filterEditor.cursor, initialText)
}

// activateFilterEditorField applies the action for whichever field is
// currently selected: text fields (description/regex) start an inline edit,
// checkboxes toggle immediately, and the color field opens the color picker.
func (m model) activateFilterEditorField() (tea.Model, tea.Cmd) {
	if len(m.filters.Filters) == 0 {
		m.editingFilter = false
		return m, nil
	}
	filter := &m.filters.Filters[m.filters.Cursor]

	switch m.filterEditor.cursor {
	case fieldDescription:
		m.filterEditor.editingText = true
		m.filterEditor.textBuf = filter.XML.Description

	case fieldRegex:
		m.filterEditor.editingText = true
		m.filterEditor.textBuf = filter.XML.Text
		m.filterEditor.regexErr = ""

	case fieldCaseSensitive:
		filter.CaseSensitive = !filter.CaseSensitive
		if regex, err := filterfiles.CompileRegex(filter.XML.Text, filter.CaseSensitive); err == nil {
			filter.Regex = regex
		}
		m.filtersDirty = true
		m.saveStatus = ""

	case fieldExcluding:
		filter.Excluding = !filter.Excluding
		m.filtersDirty = true
		m.saveStatus = ""

	case fieldEnabled:
		filter.IsEnabled = !filter.IsEnabled
		m.filtersDirty = true
		m.saveStatus = ""

	case fieldColor:
		m.filterEditor.colorPicker = colorPickerState{
			open:   true,
			cursor: colorPaletteIndexFor(filter.BackColor),
		}
	}

	return m, nil
}

// updateFilterEditorTextInput handles key presses while typing into the
// description or regex field, mirroring updateSearchInput's free-typing +
// enter-to-confirm / esc-to-discard pattern. A regex that fails to compile
// stays in edit mode with regexErr set (rendered by renderFilterEditor)
// rather than being discarded, so the user can see why and fix it. ctrl+e
// suspends the UI and hands textBuf to $EDITOR, for fields too long to
// comfortably type on one terminal line (see openFilterFieldEditorCmd).
func (m model) updateFilterEditorTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterEditor.editingText = false
		m.filterEditor.textBuf = ""
		m.filterEditor.regexErr = ""

	case "enter":
		m.commitFilterEditorTextField()

	case "ctrl+e":
		return m, openFilterFieldEditorCmd(m.filterEditor.cursor, m.filterEditor.textBuf)

	case "backspace":
		if len(m.filterEditor.textBuf) > 0 {
			r := []rune(m.filterEditor.textBuf)
			m.filterEditor.textBuf = string(r[:len(r)-1])
		}

	case " ":
		m.filterEditor.textBuf += " "

	default:
		if len(msg.Runes) > 0 {
			m.filterEditor.textBuf += string(msg.Runes)
		}
	}

	return m, nil
}

// commitFilterEditorTextField applies m.filterEditor.textBuf as the new
// value of whichever text field m.filterEditor.cursor points at
// (fieldDescription or fieldRegex). It's used by plain enter, by ctrl+e's
// external-editor return path from mid-edit, and by ctrl+e's return path
// from a hovered (not yet being edited) row -- see filterFieldEditorFinishedMsg
// and openExternalEditorForHoveredField -- so all three behave identically.
// A regex that fails to compile drops into (or stays in) edit mode with
// regexErr set instead of being silently discarded, so the user can see why
// and fix it rather than losing what they typed -- including when ctrl+e
// was pressed from hover, which never set editingText itself.
func (m *model) commitFilterEditorTextField() {
	filter := &m.filters.Filters[m.filters.Cursor]
	switch m.filterEditor.cursor {
	case fieldDescription:
		filter.XML.Description = m.filterEditor.textBuf
		m.filtersDirty = true
		m.saveStatus = ""
		m.filterEditor.editingText = false
		m.filterEditor.textBuf = ""

	case fieldRegex:
		regex, err := filterfiles.CompileRegex(m.filterEditor.textBuf, filter.CaseSensitive)
		if err != nil {
			m.filterEditor.editingText = true
			m.filterEditor.regexErr = err.Error()
			return
		}
		filter.XML.Text = m.filterEditor.textBuf
		filter.Regex = regex
		m.filtersDirty = true
		m.saveStatus = ""
		m.filterEditor.editingText = false
		m.filterEditor.textBuf = ""
		m.filterEditor.regexErr = ""
	}
}

// filterFieldEditorFinishedMsg is sent once the external $EDITOR process
// invoked from a filter editor text field (ctrl+e) has exited.
type filterFieldEditorFinishedMsg struct {
	err      error
	tempFile string
	field    filterEditorField
}

// openFilterFieldEditorCmd writes initialText to a temp file and opens it in
// the user's $EDITOR (falling back to "vi"), suspending the UI while the
// editor runs -- for filling in a filter editor text field (description or
// regex) with more room than a single terminal line offers.
func openFilterFieldEditorCmd(field filterEditorField, initialText string) tea.Cmd {
	tmpFile, err := os.CreateTemp("", "skim-filter-field-*.txt")
	if err != nil {
		return func() tea.Msg {
			return filterFieldEditorFinishedMsg{err: err, field: field}
		}
	}

	if _, err := tmpFile.WriteString(initialText); err != nil {
		tmpFile.Close()
		return func() tea.Msg {
			return filterFieldEditorFinishedMsg{err: err, field: field}
		}
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return filterFieldEditorFinishedMsg{err: err, tempFile: tmpFile.Name(), field: field}
	})
}

// renderFilterEditor draws the filter editor form: one row per field, with
// the row under the cursor marked, text fields showing their in-progress
// buffer while being edited, and the color field showing a live swatch.
func (m model) renderFilterEditor() string {
	if m.filterEditor.colorPicker.open {
		return m.renderColorPicker()
	}

	if len(m.filters.Filters) == 0 {
		return baseStyle.Render("No filter selected. esc to close.")
	}
	filter := m.filters.Filters[m.filters.Cursor]

	header := "Edit Filter  —  up/down: select field   enter: edit/toggle   esc: close"
	switch {
	case m.filterEditor.editingText:
		header = "Edit Filter  —  enter: confirm   ctrl+e: edit in $EDITOR   esc: discard"
	case m.filterEditor.cursor == fieldDescription || m.filterEditor.cursor == fieldRegex:
		header = "Edit Filter  —  up/down: select field   enter: edit   ctrl+e: edit in $EDITOR   esc: close"
	}

	var b strings.Builder
	b.WriteString(header + "\n\n")

	rows := []struct {
		field filterEditorField
		label string
	}{
		{fieldDescription, "Description"},
		{fieldRegex, "Regex"},
		{fieldCaseSensitive, "Case sensitive"},
		{fieldExcluding, "Excluding"},
		{fieldEnabled, "Enabled"},
		{fieldColor, "Color"},
	}

	for _, row := range rows {
		cursor := "  "
		if row.field == m.filterEditor.cursor {
			cursor = "> "
		}

		var value string
		switch row.field {
		case fieldDescription:
			value = m.renderFilterEditorTextValue(fieldDescription, filter.XML.Description)
		case fieldRegex:
			value = m.renderFilterEditorTextValue(fieldRegex, filter.XML.Text)
			if m.filterEditor.editingText && m.filterEditor.cursor == fieldRegex && m.filterEditor.regexErr != "" {
				value += fmt.Sprintf("  (invalid regex: %s)", m.filterEditor.regexErr)
			}
		case fieldCaseSensitive:
			value = renderFilterEditorCheckbox(filter.CaseSensitive)
		case fieldExcluding:
			value = renderFilterEditorCheckbox(filter.Excluding)
		case fieldEnabled:
			value = renderFilterEditorCheckbox(filter.IsEnabled)
		case fieldColor:
			swatch := lipgloss.NewStyle().Background(lipgloss.Color(filter.BackColor)).Render("    ")
			value = fmt.Sprintf("%s %s", swatch, filter.BackColor)
		}

		b.WriteString(fmt.Sprintf("%s%-16s%s\n", cursor, row.label+":", value))
	}

	return baseStyle.Render(b.String())
}

// renderFilterEditorTextValue renders a text field's bracketed value: the
// in-progress buffer while it's being edited, otherwise its last-committed
// value. It deliberately doesn't render its own cursor glyph -- like
// renderSearchPrompt/renderJumpLinePrompt, it relies on the terminal's own
// cursor for that (a synthetic mark here previously used U+258F, which
// several terminal/font combinations render as blank space, making it look
// like a stray trailing space in the buffer).
func (m model) renderFilterEditorTextValue(field filterEditorField, committed string) string {
	if m.filterEditor.editingText && m.filterEditor.cursor == field {
		return "[" + m.filterEditor.textBuf + "]"
	}
	return "[" + committed + "]"
}

func renderFilterEditorCheckbox(checked bool) string {
	if checked {
		return "[x]"
	}
	return "[ ]"
}
