package engine

import (
	"sync"
	"time"

	"github.com/dixiedream/metro/internal/audio"
	"github.com/dixiedream/metro/internal/model"
)

// BeatEvent is emitted on each scheduled click (including silent ones for UI).
type BeatEvent struct {
	Bar           int
	Beat          int // 0-based within bar
	Subdiv        int // 0-based within beat
	Kind          model.BeatKind
	Level         audio.ClickLevel
	Silent        bool // coach mute or beat silent — no sound
	BPM           float64
	TrackerDone   bool
	CoachSilent   bool
}

type Engine struct {
	mu sync.Mutex

	player *audio.Player
	cfg    model.Config
	track  model.Tracker
	auto   model.Automator
	coach  model.Coach

	playing bool
	stopCh  chan struct{}
	events  chan BeatEvent

	barCount    int
	beatInBar   int
	startTime   time.Time
	playSeconds int

	autoBarAccum int
	autoSecAccum int
	coachBarPos  int // position in play+silent cycle
}

func New(player *audio.Player) *Engine {
	return &Engine{
		player: player,
		cfg:    model.DefaultConfig(),
		track:  model.DefaultTracker(),
		auto:   model.DefaultAutomator(),
		coach:  model.DefaultCoach(),
		events: make(chan BeatEvent, 64),
	}
}

func (e *Engine) Events() <-chan BeatEvent { return e.events }

func (e *Engine) Config() model.Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cfg
}

func (e *Engine) SetConfig(c model.Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	c.Clamp()
	e.cfg = c
	if e.player != nil {
		e.player.SetSoundSet(c.SoundSet)
		e.player.SetVolume(c.Volume)
		e.player.SetMuted(c.Muted)
	}
}

func (e *Engine) Tracker() model.Tracker {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.track
	t.Elapsed = e.trackerElapsedLocked()
	return t
}

func (e *Engine) SetTracker(t model.Tracker) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.track = t
}

func (e *Engine) Automator() model.Automator {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.auto
}

func (e *Engine) SetAutomator(a model.Automator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if a.EveryN < 1 {
		a.EveryN = 1
	}
	e.auto = a
}

func (e *Engine) Coach() model.Coach {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.coach
}

func (e *Engine) SetCoach(c model.Coach) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c.PlayBars < 1 {
		c.PlayBars = 1
	}
	if c.SilentBars < 1 {
		c.SilentBars = 1
	}
	e.coach = c
}

func (e *Engine) IsPlaying() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playing
}

func (e *Engine) Play() {
	e.mu.Lock()
	if e.playing {
		e.mu.Unlock()
		return
	}
	e.playing = true
	e.stopCh = make(chan struct{})
	e.barCount = 0
	e.beatInBar = 0
	e.playSeconds = 0
	e.autoBarAccum = 0
	e.autoSecAccum = 0
	e.coachBarPos = 0
	e.track.Elapsed = 0
	e.startTime = time.Now()
	stop := e.stopCh
	e.mu.Unlock()
	go e.loop(stop)
}

func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.playing {
		e.mu.Unlock()
		return
	}
	e.playing = false
	ch := e.stopCh
	e.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (e *Engine) Toggle() {
	if e.IsPlaying() {
		e.Stop()
	} else {
		e.Play()
	}
}

func (e *Engine) AdjustBPM(delta float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.BPM += delta
	e.cfg.Clamp()
}

func (e *Engine) SetBPM(bpm float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.BPM = bpm
	e.cfg.Clamp()
}

func (e *Engine) trackerElapsedLocked() int {
	switch e.track.Mode {
	case model.TrackerBars:
		return e.barCount
	case model.TrackerTime:
		if e.playing {
			return int(time.Since(e.startTime).Seconds())
		}
		return e.playSeconds
	default:
		return 0
	}
}

func (e *Engine) loop(stop <-chan struct{}) {
	e.mu.Lock()
	start := time.Now()
	e.startTime = start
	tickIndex := 0
	e.mu.Unlock()

	for {
		select {
		case <-stop:
			e.mu.Lock()
			e.playSeconds = int(time.Since(e.startTime).Seconds())
			e.playing = false
			e.mu.Unlock()
			return
		default:
		}

		e.mu.Lock()
		cfg := e.cfg
		bar := e.barCount
		beat := e.beatInBar
		e.mu.Unlock()

		pattern := model.PatternByID(cfg.PatternID)
		if len(pattern.Subdivs) == 0 {
			pattern = model.PatternByID("quarter")
		}

		// Build absolute schedule for current beat's subdivisions.
		beatDur := beatDuration(cfg.BPM, cfg.Denominator)
		// For compound meters with dotted-quarter feel, beat is still numerator pulses of eighths
		// grouped by pattern; we treat each numerator count as one "beat" of length beatDur.
		offsets := make([]time.Duration, len(pattern.Subdivs))
		var acc float64
		for i, frac := range pattern.Subdivs {
			offsets[i] = time.Duration(acc * float64(beatDur))
			acc += frac
		}

		// Absolute time base for this beat within the whole run.
		// Total beats so far: bar*numerator + beat, each of beatDur.
		beatsSoFar := bar*cfg.Numerator + beat
		beatStart := start.Add(time.Duration(beatsSoFar) * beatDur)

		for si, off := range offsets {
			target := beatStart.Add(off)
			wait := time.Until(target)
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-stop:
					timer.Stop()
					e.mu.Lock()
					e.playSeconds = int(time.Since(e.startTime).Seconds())
					e.playing = false
					e.mu.Unlock()
					return
				case <-timer.C:
				}
			}

			e.mu.Lock()
			cfg = e.cfg
			coach := e.coach
			kind := model.BeatRegular
			if beat < len(cfg.Beats) {
				kind = cfg.Beats[beat]
			}
			coachSilent := false
			if coach.Enabled {
				cycle := coach.PlayBars + coach.SilentBars
				pos := e.coachBarPos % cycle
				if pos >= coach.PlayBars {
					coachSilent = true
				}
			}
			silent := kind == model.BeatSilent || coachSilent || cfg.Muted
			level := audio.ClickRegular
			if si == 0 {
				switch kind {
				case model.BeatAccent:
					level = audio.ClickAccent
				case model.BeatRegular:
					level = audio.ClickRegular
				}
			} else {
				level = audio.ClickSoft
				if !pattern.SubAccents[si] {
					// soft subdiv
				}
			}
			ev := BeatEvent{
				Bar:         e.barCount,
				Beat:        beat,
				Subdiv:      si,
				Kind:        kind,
				Level:       level,
				Silent:      silent,
				BPM:         cfg.BPM,
				CoachSilent: coachSilent,
			}
			e.mu.Unlock()

			if !silent && si == 0 && kind != model.BeatSilent {
				e.player.Play(level)
			} else if !silent && si > 0 {
				e.player.Play(audio.ClickSoft)
			}

			select {
			case e.events <- ev:
			default:
			}
			_ = tickIndex
			tickIndex++
		}

		// Advance beat / bar
		e.mu.Lock()
		e.beatInBar++
		finishedBar := false
		if e.beatInBar >= e.cfg.Numerator {
			e.beatInBar = 0
			e.barCount++
			finishedBar = true
			if e.coach.Enabled {
				e.coachBarPos++
			}
			e.autoBarAccum++
		}

		// Tracker
		trackerDone := false
		switch e.track.Mode {
		case model.TrackerBars:
			e.track.Elapsed = e.barCount
			if e.track.Limit > 0 && e.barCount >= e.track.Limit {
				trackerDone = true
			}
		case model.TrackerTime:
			secs := int(time.Since(e.startTime).Seconds())
			e.track.Elapsed = secs
			e.playSeconds = secs
			if e.track.Limit > 0 && secs >= e.track.Limit {
				trackerDone = true
			}
		}

		// Automator
		if e.auto.Enabled && finishedBar {
			if e.auto.Unit == model.UnitBars && e.auto.EveryN > 0 && e.autoBarAccum >= e.auto.EveryN {
				e.cfg.BPM += e.auto.DeltaBPM
				e.cfg.Clamp()
				e.autoBarAccum = 0
			}
		}
		if e.auto.Enabled && e.auto.Unit == model.UnitSeconds {
			secs := int(time.Since(e.startTime).Seconds())
			if e.auto.EveryN > 0 && secs/e.auto.EveryN > e.autoSecAccum {
				e.autoSecAccum = secs / e.auto.EveryN
				e.cfg.BPM += e.auto.DeltaBPM
				e.cfg.Clamp()
			}
		}

		shouldStop := trackerDone && e.track.AutoStop && e.track.Mode != model.TrackerOff
		if shouldStop {
			select {
			case e.events <- BeatEvent{TrackerDone: true, BPM: e.cfg.BPM, Bar: e.barCount}:
			default:
			}
			e.playing = false
			e.mu.Unlock()
			return
		}
		e.mu.Unlock()
	}
}

// beatDuration returns the duration of one counted beat.
// BPM is always quarter-note BPM in Tempo-style apps; adjust by denominator.
func beatDuration(bpm float64, denominator int) time.Duration {
	if bpm < model.MinBPM {
		bpm = model.MinBPM
	}
	quarter := time.Minute / time.Duration(bpm)
	switch denominator {
	case 2:
		return quarter * 2 // half note
	case 8:
		return quarter / 2 // eighth note
	default:
		return quarter
	}
}
