package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
)

// Default ANSI palette (matches termenv ansiHex indices 0–15).
var defaultANSI = [16]string{
	"#000000", "#800000", "#008000", "#808000",
	"#000080", "#800080", "#008080", "#c0c0c0",
	"#808080", "#ff0000", "#00ff00", "#ffff00",
	"#0000ff", "#ff00ff", "#00ffff", "#ffffff",
}

type Palette struct {
	Foreground string
	Background string
	ANSI       [16]string
}

func (p Palette) color(i int) string {
	if i >= 0 && i < 16 && p.ANSI[i] != "" {
		return p.ANSI[i]
	}
	return defaultANSI[i]
}

func (p Palette) hasData() bool {
	if p.Foreground != "" || p.Background != "" {
		return true
	}
	for _, c := range p.ANSI {
		if c != "" {
			return true
		}
	}
	return false
}

var (
	themeOnce sync.Once

	titleStyle       lipgloss.Style
	dimStyle         lipgloss.Style
	activeStyle      lipgloss.Style
	boxStyle         lipgloss.Style
	muteStyle        lipgloss.Style
	playStyle        lipgloss.Style
	pauseStyle       lipgloss.Style
	beatAccentStyle  lipgloss.Style
	beatRegularStyle lipgloss.Style
	beatSilentStyle  lipgloss.Style
	beatFlashStyle   lipgloss.Style
	coachSilentStyle lipgloss.Style
	statusStyle      lipgloss.Style
	logoNoteStyle    lipgloss.Style
	hintKeyStyle     lipgloss.Style
)

func initTheme() {
	themeOnce.Do(func() {
		initMonochromeStyles()
		if termenv.EnvNoColor() {
			return
		}
		p := LoadPalette()
		if !p.hasData() {
			return
		}
		p = p.finalize()
		applyPalette(p)
	})
}

func initMonochromeStyles() {
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle = lipgloss.NewStyle().Faint(true)
	activeStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	boxStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	muteStyle = titleStyle
	playStyle = titleStyle
	pauseStyle = titleStyle
	beatAccentStyle = titleStyle
	beatRegularStyle = titleStyle
	beatSilentStyle = dimStyle
	beatFlashStyle = activeStyle
	coachSilentStyle = dimStyle
	statusStyle = titleStyle
	logoNoteStyle = titleStyle
	hintKeyStyle = titleStyle
}

func applyPalette(p Palette) {
	dark := terminalDarkBackground(p.Background)

	fg := p.Foreground
	if dark {
		fg = firstNonEmpty(fg, p.color(15), p.color(7))
	} else {
		fg = firstNonEmpty(fg, p.color(0), p.color(8))
	}
	fg = ensureReadable(fg, p.Background, dark, p)

	var cyan, red, green, yellow, magenta, dim, border string
	if dark {
		cyan = firstNonEmpty(p.color(14), p.color(6), p.color(12))
		red = firstNonEmpty(p.color(9), p.color(1))
		green = firstNonEmpty(p.color(10), p.color(2))
		yellow = firstNonEmpty(p.color(11), p.color(3))
		magenta = firstNonEmpty(p.color(13), p.color(5))
		dim = firstNonEmpty(p.color(7), p.color(8))
	} else {
		cyan = firstNonEmpty(p.color(6), p.color(14), p.color(4))
		red = firstNonEmpty(p.color(1), p.color(9))
		green = firstNonEmpty(p.color(2), p.color(10))
		yellow = firstNonEmpty(p.color(3), p.color(11))
		magenta = firstNonEmpty(p.color(5), p.color(13))
		dim = firstNonEmpty(p.color(8), p.color(0))
	}
	dim = ensureReadable(dim, p.Background, dark, p)
	border = dim

	var beatAccent, beatSilent, beatFlash int
	if dark {
		beatAccent, beatSilent, beatFlash = 9, 8, 11
	} else {
		beatAccent, beatSilent, beatFlash = 1, 8, 3
	}

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipglossColor(fg))
	dimStyle = lipgloss.NewStyle().Faint(true).Foreground(lipglossColor(dim))
	activeStyle = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipglossColor(cyan))
	muteStyle = lipgloss.NewStyle().Bold(true).Foreground(lipglossColor(red))
	playStyle = lipgloss.NewStyle().Foreground(lipglossColor(green))
	pauseStyle = lipgloss.NewStyle().Foreground(lipglossColor(fg))
	// Beat row uses ANSI palette slots so the terminal's own colors render correctly.
	beatAccentStyle = lipgloss.NewStyle().Foreground(ansiIdx(beatAccent))
	beatRegularStyle = lipgloss.NewStyle()
	beatSilentStyle = lipgloss.NewStyle().Foreground(ansiIdx(beatSilent))
	beatFlashStyle = lipgloss.NewStyle().Bold(true).Foreground(ansiIdx(beatFlash))
	coachSilentStyle = lipgloss.NewStyle().Faint(true).Foreground(lipglossColor(magenta))
	statusStyle = lipgloss.NewStyle().Foreground(lipglossColor(yellow))
	logoNoteStyle = lipgloss.NewStyle().Foreground(lipglossColor(green))
	hintKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipglossColor(cyan))

	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipglossColor(border)).
		Padding(0, 1)
}

func ansiIdx(i int) lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("%d", i))
}

func terminalDarkBackground(paletteBG string) bool {
	if termBG := termenvColorHex(termenv.BackgroundColor()); termBG != "" {
		return colorDark(termBG)
	}
	if paletteBG != "" {
		return colorDark(paletteBG)
	}
	return termenv.HasDarkBackground()
}

func colorDark(hex string) bool {
	c, err := colorful.Hex(hex)
	if err != nil {
		return true
	}
	_, _, l := c.Hsl()
	return l < 0.5
}

// ensureReadable picks a fallback from the palette when fg is too close to the terminal bg.
func ensureReadable(fg, bg string, dark bool, p Palette) string {
	if fg == "" {
		return fg
	}
	if bg == "" || readableOn(fg, bg) {
		return fg
	}
	if dark {
		return firstNonEmpty(p.color(15), p.color(7), fg)
	}
	return firstNonEmpty(p.color(0), p.color(8), fg)
}

func readableOn(fg, bg string) bool {
	f, err1 := colorful.Hex(fg)
	b, err2 := colorful.Hex(bg)
	if err1 != nil || err2 != nil {
		return true
	}
	_, _, lf := f.Hsl()
	_, _, lb := b.Hsl()
	return abs(lf-lb) > 0.25
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func lipglossColor(hex string) lipgloss.Color {
	return lipgloss.Color(hex)
}

// LoadPalette resolves colors from xrdb, Xresources files, then termenv defaults.
func LoadPalette() Palette {
	var p Palette
	if data, err := queryXRDB(); err == nil {
		mergeResources(&p, data)
	}
	for _, path := range xresourcesPaths() {
		if data, err := os.ReadFile(path); err == nil {
			mergeResources(&p, data)
		}
	}
	return p
}

func (p Palette) finalize() Palette {
	out := p
	for i := range out.ANSI {
		if out.ANSI[i] == "" {
			out.ANSI[i] = defaultANSI[i]
		}
	}

	// Terminal fg/bg reflect what is actually on screen (Wayland, etc.).
	termFG := termenvColorHex(termenv.ForegroundColor())
	termBG := termenvColorHex(termenv.BackgroundColor())
	if termFG != "" {
		out.Foreground = termFG
	} else if out.Foreground == "" {
		if colorDark(firstNonEmpty(out.Background, termBG, out.color(0))) {
			out.Foreground = firstNonEmpty(out.color(15), out.color(7))
		} else {
			out.Foreground = firstNonEmpty(out.color(0), out.color(8))
		}
	}
	if termBG != "" {
		out.Background = termBG
	} else if out.Background == "" {
		if colorDark(out.Foreground) {
			out.Background = firstNonEmpty(out.color(0), "#000000")
		} else {
			out.Background = firstNonEmpty(out.color(15), "#ffffff")
		}
	}
	return out
}

func termenvColorHex(c termenv.Color) string {
	if _, ok := c.(termenv.NoColor); ok {
		return ""
	}
	ch := termenv.ConvertToRGB(c)
	r, g, b := ch.RGB255()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func queryXRDB() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "xrdb", "-query")
	return cmd.Output()
}

func xresourcesPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".Xresources"),
		filepath.Join(home, ".config", "Xresources"),
	}
}

var resourceLine = regexp.MustCompile(`^[^!].*?(color(1[0-5]|[0-9])|foreground|background)\s*:\s*(.+)$`)

func mergeResources(p *Palette, data []byte) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		m := resourceLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := m[1]
		raw := strings.TrimSpace(m[3])
		hex, ok := parseColorSpec(raw)
		if !ok {
			continue
		}
		switch {
		case key == "foreground":
			p.Foreground = hex
		case key == "background":
			p.Background = hex
		case strings.HasPrefix(key, "color"):
			var idx int
			if _, err := fmt.Sscanf(strings.TrimPrefix(key, "color"), "%d", &idx); err == nil && idx >= 0 && idx < 16 {
				p.ANSI[idx] = hex
			}
		}
	}
}

func parseColorSpec(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "#") {
		return normalizeHex(raw)
	}
	if strings.HasPrefix(raw, "rgb:") {
		return parseRGBSpec(raw)
	}
	// Some resources use names without #; try as hex anyway.
	if len(raw) == 6 || len(raw) == 3 {
		return normalizeHex("#" + raw)
	}
	return "", false
}

func parseRGBSpec(s string) (string, bool) {
	s = strings.TrimPrefix(s, "rgb:")
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return "", false
	}
	toByte := func(part string) (byte, bool) {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, false
		}
		var v int
		switch len(part) {
		case 1:
			v = int(part[0]-'0') * 0x11
			if part[0] >= 'a' && part[0] <= 'f' {
				v = int(part[0]-'a'+10) * 0x11
			} else if part[0] >= 'A' && part[0] <= 'F' {
				v = int(part[0]-'A'+10) * 0x11
			}
		case 2:
			_, err := fmt.Sscanf(part, "%x", &v)
			if err != nil {
				return 0, false
			}
		case 4:
			var hi int
			if _, err := fmt.Sscanf(part[:2], "%x", &hi); err != nil {
				return 0, false
			}
			v = hi
		default:
			return 0, false
		}
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return byte(v), true
	}
	r, ok1 := toByte(parts[0])
	g, ok2 := toByte(parts[1])
	b, ok3 := toByte(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return "", false
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, b), true
}

func normalizeHex(h string) (string, bool) {
	h = strings.TrimPrefix(strings.TrimSpace(h), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return "", false
	}
	c, err := colorful.Hex("#" + h)
	if err != nil {
		return "", false
	}
	r, g, b := c.RGB255()
	return fmt.Sprintf("#%02x%02x%02x", r, g, b), true
}
