package model_test

import (
	"testing"

	"github.com/ale/metro/internal/model"
)

func TestTempoMarking(t *testing.T) {
	cases := []struct {
		bpm  float64
		want string
	}{
		{39, "Grave"},
		{60, "Adagio"},
		{120, "Allegro"},
		{200, "Prestissimo"},
	}
	for _, c := range cases {
		if got := model.TempoMarking(c.bpm); got != c.want {
			t.Errorf("TempoMarking(%.0f)=%q want %q", c.bpm, got, c.want)
		}
	}
}

func TestValidSignaturesCount(t *testing.T) {
	sigs := model.ValidSignatures()
	// 13/2 + 13/4 + 6 eighth signatures = 32
	if len(sigs) != 32 {
		t.Fatalf("got %d signatures, want 32", len(sigs))
	}
}

func TestNextPrevSignature(t *testing.T) {
	n, d := model.NextSignature(4, 4)
	if n == 4 && d == 4 {
		t.Fatal("NextSignature did not advance")
	}
	n2, d2 := model.PrevSignature(n, d)
	if n2 != 4 || d2 != 4 {
		t.Fatalf("PrevSignature got %d/%d want 4/4", n2, d2)
	}
}

func TestBeatCycle(t *testing.T) {
	b := model.BeatAccent
	b = b.Cycle()
	if b != model.BeatRegular {
		t.Fatal(b)
	}
	b = b.Cycle()
	if b != model.BeatSilent {
		t.Fatal(b)
	}
	b = b.Cycle()
	if b != model.BeatAccent {
		t.Fatal(b)
	}
}

func TestPatterns(t *testing.T) {
	if len(model.PatternsFor(false)) != 6 {
		t.Fatal("expected 6 simple patterns")
	}
	if len(model.PatternsFor(true)) != 3 {
		t.Fatal("expected 3 compound patterns")
	}
}
