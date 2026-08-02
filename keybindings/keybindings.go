// Package keybindings defines skim's rebindable actions and loads/saves the
// user's key assignments to a JSON config file on disk.
package keybindings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Action string

const (
	Quit                  Action = "quit"
	CursorUp              Action = "cursor_up"
	CursorDown            Action = "cursor_down"
	CursorLeft            Action = "cursor_left"
	CursorRight           Action = "cursor_right"
	Toggle                Action = "toggle"
	SwitchFocus           Action = "switch_focus"
	ToggleHideUnmatched   Action = "toggle_hide_unmatched"
	EditRegex             Action = "edit_regex"
	OpenKeybindingsScreen Action = "open_keybindings"
	Search                Action = "search"
	SearchNext            Action = "search_next"
	SearchPrev            Action = "search_prev"
)

// Scope limits which focused view an action's keys are considered in.
// ScopeGlobal actions are checked in every focus; ScopeFilterView /
// ScopeLogView actions are only checked while that view is focused, and are
// checked before ScopeGlobal so they can safely reuse the same default key
// as a global action without colliding (e.g. "h" moves the filter column
// selection in the Filters view, but hides unmatched lines in the Log view).
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeFilterView
	ScopeLogView
)

type Spec struct {
	Action      Action
	Scope       Scope
	Description string
	DefaultKeys []string
}

// Registry is the ordered, canonical list of all rebindable actions.
var Registry = []Spec{
	{Quit, ScopeGlobal, "quit", []string{"ctrl+c", "q"}},
	{CursorUp, ScopeGlobal, "move cursor up", []string{"up", "k"}},
	{CursorDown, ScopeGlobal, "move cursor down", []string{"down", "j"}},
	{CursorLeft, ScopeGlobal, "move column left", []string{"left", "h"}},
	{CursorRight, ScopeGlobal, "move column right", []string{"right", "l"}},
	{Toggle, ScopeGlobal, "toggle selection", []string{"enter", " "}},
	{SwitchFocus, ScopeGlobal, "switch focus", []string{"tab"}},
	{ToggleHideUnmatched, ScopeLogView, "hide unmatched lines", []string{"h"}},
	{EditRegex, ScopeFilterView, "edit regex in $EDITOR", []string{"i"}},
	{OpenKeybindingsScreen, ScopeGlobal, "edit keybindings", []string{"K"}},
	{Search, ScopeLogView, "search log", []string{"/"}},
	{SearchNext, ScopeLogView, "jump to next match", []string{"n"}},
	{SearchPrev, ScopeLogView, "jump to previous match", []string{"N"}},
}

// SpecFor returns the registry entry for an action.
func SpecFor(action Action) (Spec, bool) {
	for _, spec := range Registry {
		if spec.Action == action {
			return spec, true
		}
	}
	return Spec{}, false
}

type KeyMap map[Action][]string

// Defaults returns a fresh KeyMap populated with each action's default keys.
func Defaults() KeyMap {
	km := make(KeyMap, len(Registry))
	for _, spec := range Registry {
		keys := make([]string, len(spec.DefaultKeys))
		copy(keys, spec.DefaultKeys)
		km[spec.Action] = keys
	}
	return km
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skim", "keybindings.json"), nil
}

// Load returns the effective keymap: defaults overridden by any bindings
// found in the user's on-disk config file. If the config file or config
// directory can't be resolved or doesn't exist yet, defaults are returned.
func Load() (KeyMap, error) {
	km := Defaults()

	path, err := configPath()
	if err != nil {
		return km, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return km, nil
		}
		return km, err
	}

	var stored map[Action][]string
	if err := json.Unmarshal(data, &stored); err != nil {
		return km, err
	}
	for action, keys := range stored {
		km[action] = keys
	}
	return km, nil
}

// Save persists the given keymap to the user's config file, creating the
// config directory if needed.
func Save(km KeyMap) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(km, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
