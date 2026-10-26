// Package ui holds every terminal style gh fleet uses: lipgloss styles
// built from the brand palette, the huh theme for prompts and the slog
// handler. Nothing else in the program defines colors.
package ui

import (
	"errors"
	"image/color"
	"io"
	"log/slog"
	"os"
	"sync"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/x/term"
)

// Adaptive is a color with a variant for light and for dark terminals.
type Adaptive struct {
	Light, Dark color.Color
}

// For picks the variant for a dark or light background.
func (a Adaptive) For(dark bool) color.Color {
	if dark {
		return a.Dark
	}
	return a.Light
}

// Brand palette, as adaptive colors that read on light and dark terminals.
var (
	Brand = Adaptive{Light: lipgloss.Color("#0B7A75"), Dark: lipgloss.Color("#3DDBD9")}
	OK    = Adaptive{Light: lipgloss.Color("#1A7F37"), Dark: lipgloss.Color("#56D364")}
	Warn  = Adaptive{Light: lipgloss.Color("#9A6700"), Dark: lipgloss.Color("#E3B341")}
	Bad   = Adaptive{Light: lipgloss.Color("#CF222E"), Dark: lipgloss.Color("#FF7B72")}
	Info  = Adaptive{Light: lipgloss.Color("#0969DA"), Dark: lipgloss.Color("#79C0FF")}
	Dim   = Adaptive{Light: lipgloss.Color("#656D76"), Dark: lipgloss.Color("#8B949E")}
)

// darkBackground asks the terminal for its background color, once, and
// only when stdin and stderr are both terminals. Asking otherwise can
// block: on Windows, lipgloss opens the console itself and waits for a
// reply, which hung test binaries that imported this package when the
// query ran at startup. Without an answer, it assumes dark.
var darkBackground = sync.OnceValue(func() bool {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stderr.Fd()) {
		return true
	}
	return lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
})

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

// Color returns the styled set for the terminal's background.
func Color() Styles {
	return ColorFor(darkBackground())
}

// ColorFor returns the styled set for a dark or light background.
func ColorFor(dark bool) Styles {
	s := lipgloss.NewStyle()
	return Styles{
		Bold:   s.Bold(true),
		Header: s.Bold(true).Foreground(Brand.For(dark)),
		Brand:  s.Foreground(Brand.For(dark)),
		OK:     s.Foreground(OK.For(dark)),
		Warn:   s.Foreground(Warn.For(dark)),
		Bad:    s.Foreground(Bad.For(dark)),
		Info:   s.Foreground(Info.For(dark)),
		Dim:    s.Foreground(Dim.For(dark)),
		Add:    s.Foreground(OK.For(dark)),
		Del:    s.Foreground(Bad.For(dark)),
		Hunk:   s.Foreground(Info.For(dark)),
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
		brand := Brand.For(isDark)
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
