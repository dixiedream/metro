package engine

import (
	"sync"
	"time"

	"github.com/ale/metro/internal/model"
)

// TapTempo estimates BPM from successive taps.
type TapTempo struct {
	mu     sync.Mutex
	taps   []time.Time
	window time.Duration
}

func NewTapTempo() *TapTempo {
	return &TapTempo{window: 3 * time.Second}
}

// Tap records a tap and returns the estimated BPM (0 if not enough taps).
func (t *TapTempo) Tap() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.taps = append(t.taps, now)
	// Drop old taps
	cut := now.Add(-t.window)
	i := 0
	for i < len(t.taps) && t.taps[i].Before(cut) {
		i++
	}
	t.taps = t.taps[i:]
	if len(t.taps) < 2 {
		return 0
	}
	var sum time.Duration
	for j := 1; j < len(t.taps); j++ {
		sum += t.taps[j].Sub(t.taps[j-1])
	}
	avg := sum / time.Duration(len(t.taps)-1)
	if avg <= 0 {
		return 0
	}
	bpm := float64(time.Minute) / float64(avg)
	if bpm < model.MinBPM {
		bpm = model.MinBPM
	}
	if bpm > model.MaxBPM {
		bpm = model.MaxBPM
	}
	return bpm
}

func (t *TapTempo) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.taps = nil
}
