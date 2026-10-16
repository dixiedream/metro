package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dixiedream/metro/internal/model"
)

var helpKeySplit = regexp.MustCompile(`\s{2,}`)

func renderShortcutHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	wrapped := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		wrapped = true
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	inner := renderShortcutHintInner(s)
	if wrapped {
		return dimStyle.Render("(") + inner + dimStyle.Render(")")
	}
	return inner
}

func renderShortcutHintInner(s string) string {
	groups := strings.Split(s, "  ")
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		i := strings.IndexByte(g, ' ')
		if i < 0 {
			parts = append(parts, hintKeyStyle.Render(g))
			continue
		}
		key := g[:i]
		desc := strings.TrimSpace(g[i+1:])
		parts = append(parts, hintKeyStyle.Render(key)+" "+dimStyle.Render(desc))
	}
	return strings.Join(parts, "  ")
}

func renderHelpOverlay(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = renderHelpLine(line)
	}
	return strings.Join(out, "\n")
}

func renderHelpLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line
	}
	switch trimmed {
	case "keyboard help", "Global", "Beats", "Tracker", "Automator", "Coach":
		return titleStyle.Render(trimmed)
	}
	if strings.HasPrefix(line, "  ") {
		content := strings.TrimLeft(line, " ")
		parts := helpKeySplit.Split(content, 2)
		if len(parts) == 2 {
			keyCol := strings.TrimSpace(parts[0])
			desc := strings.TrimSpace(parts[1])
			return "  " + hintKeyStyle.Render(keyCol) + "  " + dimStyle.Render(desc)
		}
	}
	if strings.HasPrefix(trimmed, "MIDI ") {
		return dimStyle.Render(line)
	}
	if strings.HasPrefix(trimmed, "Press ") {
		return renderHelpClosingLine(trimmed)
	}
	return dimStyle.Render(line)
}

func renderHelpClosingLine(s string) string {
	// "Press ? or esc to close" — highlight key tokens.
	const prefix = "Press "
	if !strings.HasPrefix(s, prefix) {
		return dimStyle.Render(s)
	}
	rest := strings.TrimPrefix(s, prefix)
	tokens := []string{"?", "esc"}
	var b strings.Builder
	b.WriteString(dimStyle.Render(prefix))
	i := 0
	for i < len(rest) {
		matched := false
		for _, tok := range tokens {
			if strings.HasPrefix(rest[i:], tok) {
				b.WriteString(hintKeyStyle.Render(tok))
				i += len(tok)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		// copy until next token or end
		next := len(rest)
		for _, tok := range tokens {
			if j := strings.Index(rest[i:], tok); j >= 0 && i+j < next {
				next = i + j
			}
		}
		b.WriteString(dimStyle.Render(rest[i:next]))
		i = next
	}
	return b.String()
}

func beatKindStyle(k model.BeatKind) lipgloss.Style {
	switch k {
	case model.BeatAccent:
		return beatAccentStyle
	case model.BeatSilent:
		return beatSilentStyle
	default:
		return beatRegularStyle
	}
}

func renderTrackerMode(mode model.TrackerMode) string {
	switch mode {
	case model.TrackerOff:
		return dimStyle.Render(mode.String())
	default:
		return statusStyle.Render(mode.String())
	}
}

func renderEnabled(on bool) string {
	if on {
		return playStyle.Render("on")
	}
	return dimStyle.Render("off")
}

func renderBoolStatus(v bool) string {
	if v {
		return playStyle.Render("true")
	}
	return dimStyle.Render("false")
}

func renderCount(n int) string {
	return statusStyle.Render(fmt.Sprintf("%d", n))
}

func renderSignedFloat(v float64) string {
	return statusStyle.Render(fmt.Sprintf("%+g", v))
}
