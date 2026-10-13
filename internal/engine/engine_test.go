package engine_test

import (
	"testing"
	"time"

	"github.com/ale/metro/internal/engine"
	"github.com/ale/metro/internal/model"
)

func TestTapTempo(t *testing.T) {
	tap := engine.NewTapTempo()
	if bpm := tap.Tap(); bpm != 0 {
		t.Fatalf("first tap should be 0, got %v", bpm)
	}
	time.Sleep(500 * time.Millisecond)
	bpm := tap.Tap()
	if bpm < 100 || bpm > 140 {
		t.Fatalf("expected ~120 BPM from 500ms interval, got %v", bpm)
	}
}

func TestBeatDurationDenom(t *testing.T) {
	// Use public Config clamp path indirectly via engine SetBPM bounds
	e := engine.New(nil)
	e.SetBPM(5)
	if e.Config().BPM != model.MinBPM {
		t.Fatalf("bpm clamp low: %v", e.Config().BPM)
	}
	e.SetBPM(999)
	if e.Config().BPM != model.MaxBPM {
		t.Fatalf("bpm clamp high: %v", e.Config().BPM)
	}
}
