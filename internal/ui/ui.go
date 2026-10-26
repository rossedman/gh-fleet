// Package ui holds every terminal style gh fleet uses: lipgloss styles
// built from the brand palette, the huh theme for prompts and the slog
// handler. Nothing else in the program defines colors.
package ui

import (
	"errors"
	"io"
	"log/slog"
	"os"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"charm.land/log/v2"
)

// Brand palette, as adaptive colors that read on light and dark terminals.
var (
	Brand = compat.AdaptiveColor{Light: lipgloss.Color("#0B7A75"), Dark: lipgloss.Color("#3DDBD9")}
	OK    = compat.AdaptiveColor{Light: lipgloss.Color("#1A7F37"), Dark: lipgloss.Color("#56D364")}
	Warn  = compat.AdaptiveColor{Light: lipgloss.Color("#9A6700"), Dark: lipgloss.Color("#E3B341")}
	Bad   = compat.AdaptiveColor{Light: lipgloss.Color("#CF222E"), Dark: lipgloss.Color("#FF7B72")}
	Info  = compat.AdaptiveColor{Light: lipgloss.Color("#0969DA"), Dark: lipgloss.Color("#79C0FF")}
	Dim   = compat.AdaptiveColor{Light: lipgloss.Color("#656D76"), Dark: lipgloss.Color("#8B949E")}
)

// Styles is the set of styles for one output stream. The zero value (from
// Plain) renders text unchanged, for pipes and NO_COLOR.
type Styles struct {
	Bold   lipgloss.Style
	Header lipgloss.Style
	Brand  lipgloss.Style
	OK     lipgloss.Style
	Warn   lipgloss.Style
	Bad    lipgloss.Style
	Info   lipgloss.Style
	Dim    lipgloss.Style
	Add    lipgloss.Style
	Del    lipgloss.Style
	Hunk   lipgloss.Style
}

// Plain returns styles that add no escape codes.
func Plain() Styles {
	s := lipgloss.NewStyle()
	return Styles{Bold: s, Header: s, Brand: s, OK: s, Warn: s, Bad: s, Info: s, Dim: s, Add: s, Del: s, Hunk: s}
}

// Color returns the styled set.
func Color() Styles {
	s := lipgloss.NewStyle()
	return Styles{
		Bold:   s.Bold(true),
		Header: s.Bold(true).Foreground(Brand),
		Brand:  s.Foreground(Brand),
		OK:     s.Foreground(OK),
		Warn:   s.Foreground(Warn),
		Bad:    s.Foreground(Bad),
		Info:   s.Foreground(Info),
		Dim:    s.Foreground(Dim),
		Add:    s.Foreground(OK),
		Del:    s.Foreground(Bad),
		Hunk:   s.Foreground(Info),
	}
}

// For returns colored styles when color is true, else plain ones.
func For(color bool) Styles {
	if color {
		return Color()
	}
	return Plain()
}

// ColorEnabled reports whether output to a terminal may be colored,
// honoring NO_COLOR, CLICOLOR=0 and CLICOLOR_FORCE as gh does.
func ColorEnabled(isTTY bool) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if v := os.Getenv("CLICOLOR_FORCE"); v != "" && v != "0" {
		return true
	}
	return isTTY
}

// Accessible reports whether prompts should use huh's accessible mode,
// set with GH_FLEET_ACCESSIBLE.
func Accessible() bool {
	return os.Getenv("GH_FLEET_ACCESSIBLE") != ""
}

// Theme is the huh theme for prompts.
func Theme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)
		brand := lipgloss.LightDark(isDark)(Brand.Light, Brand.Dark)
		t.Focused.Title = t.Focused.Title.Foreground(brand).Bold(true)
		t.Focused.FocusedButton = t.Focused.FocusedButton.Background(brand)
		return t
	})
}

// ErrNotConfirmed is returned when the user declines a confirmation.
var ErrNotConfirmed = errors.New("canceled")

// Confirm asks a yes/no question on the terminal.
func Confirm(in io.Reader, out io.Writer, title string) (bool, error) {
	ok := false
	c := huh.NewConfirm().Title(title).Affirmative("Yes").Negative("No").Value(&ok)
	err := huh.NewForm(huh.NewGroup(c)).
		WithTheme(Theme()).
		WithAccessible(Accessible()).
		WithInput(in).
		WithOutput(out).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return false, nil
	}
	return ok, err
}

// NewLogger returns a slog logger writing human-facing logs to w through
// charmbracelet/log.
func NewLogger(w io.Writer, debug bool) *slog.Logger {
	l := log.NewWithOptions(w, log.Options{ReportTimestamp: false})
	if debug {
		l.SetLevel(log.DebugLevel)
	}
	return slog.New(l)
}
