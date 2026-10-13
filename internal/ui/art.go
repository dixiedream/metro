package ui

import (
	"strings"
	"time"
)

// logoWide is shown when the terminal is at least logoMinWidth columns wide.
const logoMinWidth = 44

var logoWideLines = []string{
	`   ▗▖  ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖  ▗▄▖  `,
	`   ▐▛▚▞▜▌▐▌     █  ▐▌ ▐▌▐▌ ▐▌ `,
	`   ▐▌  ▐▌▐▛▀▀▘  █  ▐▛▀▚▖▐▌ ▐▌ `,
	`   ▐▌  ▐▌▐▙▄▄▖  █  ▐▌ ▐▌▝▚▄▞▘ `,
	`  ════════════════════════════`,
	`     ○───○───○───○───○───○    `,
}

func (m *Model) logoView() string {
	note := "♩"
	if m.engine.IsPlaying() && time.Now().Before(m.flashUntil) {
		note = "♪"
	}

	if m.width > 0 && m.width < logoMinWidth {
		noteStyled := logoNoteStyle.Render(note)
		if m.engine.IsPlaying() && time.Now().Before(m.flashUntil) {
			noteStyled = beatFlashStyle.Render(note)
		}
		return noteStyled + dimStyle.Render("  METRO  ·  terminal metronome")
	}

	lines := make([]string, len(logoWideLines))
	copy(lines, logoWideLines)
	lines[0] = strings.Replace(logoWideLines[0], "♩", note, 1)

	var b strings.Builder
	for i, line := range lines {
		if i == 0 {
			b.WriteString(titleStyle.Render(line))
		} else if i >= len(lines)-2 {
			b.WriteString(dimStyle.Render(line))
		} else {
			b.WriteString(titleStyle.Render(line))
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
