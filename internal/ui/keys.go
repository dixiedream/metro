package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ale/metro/internal/model"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	cfg := m.engine.Config()
	key := msg.String()

	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		m.persist()
		return tea.Quit
	case "?":
		m.showHelp = true
		return nil
	}

	if cmd := m.handlePresetKey(msg); cmd != nil || isPresetKey(key) {
		return cmd
	}
	if cmd := m.handlePracticeKey(msg); cmd != nil || isPracticeKey(key) {
		return cmd
	}

	switch key {
	case " ":
		m.engine.Toggle()
	case "t":
		if bpm := m.tap.Tap(); bpm > 0 {
			m.engine.SetBPM(bpm)
			m.settings.Config = m.engine.Config()
			m.status = fmt.Sprintf("Tap tempo: %.0f", bpm)
			return clearStatusCmd()
		}
	case "up", "k", "+":
		m.engine.AdjustBPM(1)
		m.settings.Config = m.engine.Config()
	case "down", "j", "-":
		m.engine.AdjustBPM(-1)
		m.settings.Config = m.engine.Config()
	case "shift+up", "K":
		m.engine.AdjustBPM(5)
		m.settings.Config = m.engine.Config()
	case "shift+down", "J":
		m.engine.AdjustBPM(-5)
		m.settings.Config = m.engine.Config()
	case "m":
		cfg.Muted = !cfg.Muted
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
	case "s":
		name := m.player.NextSoundSet()
		cfg.SoundSet = name
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		m.status = "Sound: " + name
		return clearStatusCmd()
	case "S":
		name := m.player.PrevSoundSet()
		cfg.SoundSet = name
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		m.status = "Sound: " + name
		return clearStatusCmd()
	case "[":
		n, d := model.PrevSignature(cfg.Numerator, cfg.Denominator)
		cfg.Numerator, cfg.Denominator = n, d
		cfg.Beats = model.DefaultBeats(n)
		cfg.PatternID = model.DefaultPatternID(cfg.IsCompound())
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
	case "]":
		n, d := model.NextSignature(cfg.Numerator, cfg.Denominator)
		cfg.Numerator, cfg.Denominator = n, d
		cfg.Beats = model.DefaultBeats(n)
		cfg.PatternID = model.DefaultPatternID(cfg.IsCompound())
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
	case "left", "h":
		if cfg.Numerator > 1 {
			cfg.Numerator--
			cfg.Beats = model.DefaultBeats(cfg.Numerator)
			m.engine.SetConfig(cfg)
			m.settings.Config = cfg
		}
	case "right", "l":
		if cfg.Numerator < 13 {
			cfg.Numerator++
			cfg.Beats = model.DefaultBeats(cfg.Numerator)
			m.engine.SetConfig(cfg)
			m.settings.Config = cfg
		}
	case "p":
		cfg.PatternID = model.NextPatternID(cfg.PatternID, cfg.IsCompound())
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		m.status = "Pattern: " + model.PatternByID(cfg.PatternID).Name
		return clearStatusCmd()
	case "P":
		cfg.PatternID = model.PrevPatternID(cfg.PatternID, cfg.IsCompound())
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
	case "v":
		cfg.Volume = clamp(cfg.Volume+0.05, 0, 1)
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		m.settings.Volume = cfg.Volume
	case "V":
		cfg.Volume = clamp(cfg.Volume-0.05, 0, 1)
		m.engine.SetConfig(cfg)
		m.settings.Config = cfg
		m.settings.Volume = cfg.Volume
	}
	return nil
}

func isPracticeKey(key string) bool {
	switch key {
	case "r", "R", "=", "_", "u", "z", "Z", ">", "<", ".", ",", "w", "c", "C", "b", "B", "g":
		return true
	}
	return false
}

func isPresetKey(key string) bool {
	switch key {
	case "n", "N", "a":
		return true
	}
	return false
}
