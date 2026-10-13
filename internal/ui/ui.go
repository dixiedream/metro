package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ale/metro/internal/audio"
	"github.com/ale/metro/internal/engine"
	"github.com/ale/metro/internal/midi"
	"github.com/ale/metro/internal/model"
	"github.com/ale/metro/internal/store"
	"github.com/ale/metro/internal/version"
)

type tickMsg time.Time
type beatMsg engine.BeatEvent
type midiMsg midi.Event
type statusClearMsg struct{}

type Model struct {
	store  *store.Store
	engine *engine.Engine
	player *audio.Player
	tap    *engine.TapTempo
	midi   *midi.Listener

	settings model.Settings

	width  int
	height int

	// visual beat state
	currentBeat int
	flashUntil  time.Time
	coachSilent bool

	status string

	showHelp bool
	quitting bool
}

func New(st *store.Store, eng *engine.Engine, player *audio.Player, mid *midi.Listener) (*Model, error) {
	initTheme()

	settings, err := st.LoadSettings()
	if err != nil {
		return nil, err
	}

	eng.SetConfig(settings.Config)
	eng.SetTracker(settings.Tracker)
	eng.SetAutomator(settings.Automator)
	eng.SetCoach(settings.Coach)
	player.SetVolume(settings.Volume)
	player.SetSoundSet(settings.Config.SoundSet)

	m := &Model{
		store:    st,
		engine:   eng,
		player:   player,
		tap:      engine.NewTapTempo(),
		midi:     mid,
		settings: settings,
	}

	if settings.MIDI.Enabled && mid != nil {
		_ = mid.Configure(settings.MIDI)
	}
	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(listenBeats(m.engine), listenMIDI(m.midi), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func listenBeats(eng *engine.Engine) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-eng.Events()
		if !ok {
			return nil
		}
		return beatMsg(ev)
	}
}

func listenMIDI(l *midi.Listener) tea.Cmd {
	if l == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-l.Events()
		if !ok {
			return nil
		}
		return midiMsg(ev)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tickCmd()

	case beatMsg:
		ev := engine.BeatEvent(msg)
		if ev.TrackerDone {
			m.status = "Tracker limit reached — stopped"
			return m, tea.Batch(listenBeats(m.engine), clearStatusCmd())
		}
		if ev.Subdiv == 0 {
			m.currentBeat = ev.Beat
			m.flashUntil = time.Now().Add(80 * time.Millisecond)
			m.coachSilent = ev.CoachSilent
			cfg := m.engine.Config()
			m.settings.Config.BPM = cfg.BPM
		}
		return m, listenBeats(m.engine)

	case midiMsg:
		m.handleMIDI(midi.Event(msg))
		return m, listenMIDI(m.midi)

	case statusClearMsg:
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.showHelp = false
			}
			return m, nil
		}
		return m, m.handleKey(msg)
	}
	return m, nil
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusClearMsg{} })
}

func (m *Model) handleMIDI(ev midi.Event) {
	switch ev.Action {
	case midi.ActionPlayPause:
		m.engine.Toggle()
	case midi.ActionTempoUp:
		m.engine.AdjustBPM(1)
		m.settings.Config = m.engine.Config()
	case midi.ActionTempoDown:
		m.engine.AdjustBPM(-1)
		m.settings.Config = m.engine.Config()
	}
}

func (m *Model) handlePresetKey(msg tea.KeyMsg) tea.Cmd {
	cfg := m.engine.Config()
	switch msg.String() {
	case "n":
		i := (m.currentBeat + 1) % max(1, len(cfg.Beats))
		m.currentBeat = i
		return tea.Batch()
	case "N":
		i := (m.currentBeat - 1 + max(1, len(cfg.Beats))) % max(1, len(cfg.Beats))
		m.currentBeat = i
		return tea.Batch()
	case "a":
		if len(cfg.Beats) == 0 {
			return tea.Batch()
		}
		i := m.currentBeat
		if i < 0 || i >= len(cfg.Beats) {
			i = 0
		}
		cfg.Beats[i] = cfg.Beats[i].Cycle()
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		return tea.Batch()
	}
	return nil
}

func (m *Model) handlePracticeKey(msg tea.KeyMsg) tea.Cmd {
	tr := m.engine.Tracker()
	au := m.engine.Automator()
	co := m.engine.Coach()
	switch msg.String() {
	case "r":
		tr.Mode = (tr.Mode + 1) % 3
		m.engine.SetTracker(tr)
		m.settings.Tracker = tr
	case "R":
		tr.Elapsed = 0
		m.engine.SetTracker(tr)
	case "=":
		tr.Limit++
		m.engine.SetTracker(tr)
		m.settings.Tracker = tr
	case "_":
		if tr.Limit > 1 {
			tr.Limit--
		}
		m.engine.SetTracker(tr)
		m.settings.Tracker = tr
	case "u":
		tr.AutoStop = !tr.AutoStop
		m.engine.SetTracker(tr)
		m.settings.Tracker = tr
	case "z":
		au.Enabled = !au.Enabled
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case "Z":
		au.DeltaBPM = -au.DeltaBPM
		if au.DeltaBPM == 0 {
			au.DeltaBPM = 5
		}
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case ">":
		au.EveryN++
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case "<":
		if au.EveryN > 1 {
			au.EveryN--
		}
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case ".":
		au.DeltaBPM++
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case ",":
		au.DeltaBPM--
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case "w":
		if au.Unit == model.UnitBars {
			au.Unit = model.UnitSeconds
		} else {
			au.Unit = model.UnitBars
		}
		m.engine.SetAutomator(au)
		m.settings.Automator = au
	case "c":
		co.Enabled = !co.Enabled
		m.engine.SetCoach(co)
		m.settings.Coach = co
	case "C":
		co.PlayBars++
		m.engine.SetCoach(co)
		m.settings.Coach = co
	case "b":
		if co.PlayBars > 1 {
			co.PlayBars--
		}
		m.engine.SetCoach(co)
		m.settings.Coach = co
	case "B":
		co.SilentBars++
		m.engine.SetCoach(co)
		m.settings.Coach = co
	case "g":
		if co.SilentBars > 1 {
			co.SilentBars--
		}
		m.engine.SetCoach(co)
		m.settings.Coach = co
	}
	return nil
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.showHelp {
		return m.helpView()
	}

	var b strings.Builder
	b.WriteString(m.logoView())
	b.WriteString("\n\n")
	b.WriteString(m.mainView())
	b.WriteString("\n\n")
	b.WriteString(m.footerView())
	return b.String()
}

func (m *Model) transportLine() string {
	cfg := m.engine.Config()
	var state string
	if m.engine.IsPlaying() {
		state = playStyle.Copy().Bold(true).Render("●  Playing")
	} else {
		state = dimStyle.Render("○  Stopped")
	}
	mute := ""
	if cfg.Muted {
		mute = "  " + muteStyle.Render("MUTE")
	}
	return state + mute + dimStyle.Render("  ·  space")
}

func (m *Model) ledsView(large bool) string {
	cfg := m.engine.Config()
	flashing := time.Now().Before(m.flashUntil)
	parts := make([]string, len(cfg.Beats))
	for i, kind := range cfg.Beats {
		var sym string
		switch kind {
		case model.BeatAccent:
			sym = "●"
		case model.BeatSilent:
			sym = "·"
		default:
			sym = "○"
		}
		style := beatRegularStyle
		switch kind {
		case model.BeatAccent:
			style = beatAccentStyle
		case model.BeatSilent:
			style = beatSilentStyle
		}
		if flashing && i == m.currentBeat {
			style = beatFlashStyle
			if large {
				sym = "█"
			} else {
				sym = "◉"
			}
		}
		parts[i] = style.Render(sym)
	}
	extra := ""
	if m.coachSilent {
		extra = coachSilentStyle.Render("  (coach silent)")
	}
	sep := " "
	if large {
		sep = "   "
	}
	blocks := make([]string, 0, len(parts)*2-1)
	for i, part := range parts {
		if i > 0 {
			blocks = append(blocks, sep)
		}
		blocks = append(blocks, part)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...) + extra
}

func (m *Model) bpmBlock(large bool) string {
	cfg := m.engine.Config()
	marking := model.TempoMarking(cfg.BPM)
	if large {
		return fmt.Sprintf("%s\n\n  %.0f BPM\n  %s\n  %s",
			m.ledsView(true),
			cfg.BPM,
			cfg.Signature(),
			dimStyle.Render(marking),
		)
	}
	return fmt.Sprintf("%s\n%.0f BPM  %s  %s",
		m.ledsView(false),
		cfg.BPM,
		cfg.Signature(),
		dimStyle.Render(marking),
	)
}

func (m *Model) mainView() string {
	cfg := m.engine.Config()
	pat := model.PatternByID(cfg.PatternID)
	beats := make([]string, len(cfg.Beats))
	for i, k := range cfg.Beats {
		mark := " "
		if i == m.currentBeat {
			mark = activeStyle.Render(">")
		}
		beats[i] = fmt.Sprintf("%s%d:%s", mark, i+1, beatKindStyle(k).Render(k.String()))
	}
	tr := m.engine.Tracker()
	au := m.engine.Automator()
	co := m.engine.Coach()
	trackerLine := fmt.Sprintf("%s  elapsed %s / limit %s  auto-stop %s",
		renderTrackerMode(tr.Mode),
		renderCount(tr.Elapsed),
		renderCount(tr.Limit),
		renderBoolStatus(tr.AutoStop),
	)
	automatorLine := fmt.Sprintf("%s  %s BPM every %s %s",
		renderEnabled(au.Enabled),
		renderSignedFloat(au.DeltaBPM),
		renderCount(au.EveryN),
		statusStyle.Render(au.Unit.String()),
	)
	coachLine := fmt.Sprintf("%s  play %s / silent %s bars",
		renderEnabled(co.Enabled),
		renderCount(co.PlayBars),
		renderCount(co.SilentBars),
	)
	return boxStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n\nPattern: %s  %s\nSound:   %s\nVolume:  %.0f%%  %s\n\nBeats:   %s\n         %s\n\nTracker:   %s\n          %s\n\nAutomator: %s\n          %s\n\nCoach:     %s\n          %s",
		m.transportLine(),
		m.bpmBlock(false),
		pat.Name,
		renderShortcutHint("(p/P cycle)"),
		cfg.SoundSet,
		cfg.Volume*100,
		renderShortcutHint("(s/S sound  v/V volume  m mute)"),
		strings.Join(beats, "  "),
		renderShortcutHint("(n/N focus  a cycle accent/regular/silent)"),
		trackerLine,
		renderShortcutHint("(r mode  =/_ limit  u auto-stop  R reset)"),
		automatorLine,
		renderShortcutHint("(z toggle  Z flip sign  </> every  ,/. delta  w unit)"),
		coachLine,
		renderShortcutHint("(c toggle  C/b play bars  B/g silent bars)"),
	))
}

func (m *Model) footerView() string {
	if m.status != "" {
		return statusStyle.Render(m.status)
	}
	footer := renderShortcutHint("? help  q quit")
	if v := version.Short(); v != "" {
		footer += dimStyle.Render("  ·  " + v)
	}
	return footer
}

func (m *Model) helpView() string {
	return m.logoView() + "\n\n" + boxStyle.Render(renderHelpOverlay(`keyboard help

Global
  space          play / pause
  ↑↓ / j k / +-  BPM ±1   (Shift ±5)
  t              tap tempo
  [ ]            prev/next time signature
  ←→ / h l       fewer/more beats (numerator)
  p / P          next/prev rhythm pattern
  s / S          next/prev sound set
  m              mute
  v / V          volume ±5%
  q              quit

Beats
  n / N          focus next/prev beat
  a              cycle accent → regular → silent

Tracker
  r              mode (off/bars/time)
  = / _          limit ±
  u              auto-stop toggle
  R              reset elapsed

Automator
  z              on/off
  Z              flip BPM delta sign
  </>            every N ±
  , / .          delta BPM ±
  w              bars / seconds unit

Coach
  c              on/off
  C / b          play bars ±
  B / g          silent bars ±

MIDI (settings.json): enable + note numbers for play/tempo

Press ? or esc to close`))
}

func (m *Model) persist() {
	m.settings.Config = m.engine.Config()
	m.settings.Tracker = m.engine.Tracker()
	m.settings.Automator = m.engine.Automator()
	m.settings.Coach = m.engine.Coach()
	m.settings.Volume = m.settings.Config.Volume
	_ = m.store.SaveSettings(m.settings)
}
