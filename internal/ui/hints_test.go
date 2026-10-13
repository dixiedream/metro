package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	initTheme()
	hintKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00aaaa"))
	dimStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#888888"))
	m.Run()
}

func TestRenderShortcutHintGroups(t *testing.T) {
	out := renderShortcutHint("space play  t tap")
	if !strings.Contains(out, "space") || !strings.Contains(out, "play") {
		t.Fatalf("expected key and desc in output: %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI styling in output: %q", out)
	}
}

func TestRenderShortcutHintParens(t *testing.T) {
	out := renderShortcutHint("(r mode  u auto-stop)")
	if !strings.Contains(out, "r") || !strings.Contains(out, "mode") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "(") {
		t.Fatalf("expected paren styling: %q", out)
	}
}

func TestRenderHelpOverlayBindingLine(t *testing.T) {
	raw := "  space          play / pause"
	out := renderHelpLine(raw)
	if !strings.Contains(out, "space") || !strings.Contains(out, "play") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.HasPrefix(out, "  ") {
		t.Fatalf("expected indent: %q", out)
	}
}

func TestRenderHelpOverlayTitle(t *testing.T) {
	out := renderHelpLine("Global")
	if out != titleStyle.Render("Global") {
		t.Fatalf("expected title style: %q", out)
	}
}

func TestRenderHelpClosingLine(t *testing.T) {
	out := renderHelpClosingLine("Press ? or esc to close")
	if !strings.Contains(out, "?") || !strings.Contains(out, "esc") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRenderTrackerModeActive(t *testing.T) {
	out := renderTrackerMode(2) // TrackerTime
	if !strings.Contains(out, "time") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected styled output: %q", out)
	}
}

func TestRenderEnabled(t *testing.T) {
	if !strings.Contains(renderEnabled(true), "on") {
		t.Fatal("expected on")
	}
	if !strings.Contains(renderEnabled(false), "off") {
		t.Fatal("expected off")
	}
}
