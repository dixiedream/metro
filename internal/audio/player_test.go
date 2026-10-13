package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestApplyGainZero(t *testing.T) {
	pcm := []byte{0, 1, 0, 2}
	if got := applyGain(pcm, 0); got != nil {
		t.Fatalf("expected nil for zero volume, got len %d", len(got))
	}
}

func TestApplyGainHalf(t *testing.T) {
	pcm := make([]byte, 4)
	binary.LittleEndian.PutUint16(pcm[0:2], uint16(1000))
	binary.LittleEndian.PutUint16(pcm[2:4], uint16(0xF830)) // int16(-2000)

	out := applyGain(pcm, 0.5)
	var s0, s1 int16
	_ = binary.Read(bytes.NewReader(out[0:2]), binary.LittleEndian, &s0)
	_ = binary.Read(bytes.NewReader(out[2:4]), binary.LittleEndian, &s1)
	if s0 != 500 || s1 != -1000 {
		t.Fatalf("unexpected scaled samples: %d, %d", s0, s1)
	}
}

func TestEnqueueDropsSoftWhenFull(t *testing.T) {
	p := &Player{
		playCh: make(chan ClickLevel, 2),
		stop:   make(chan struct{}),
	}
	p.playCh <- ClickSoft
	p.playCh <- ClickSoft

	p.enqueueClick(ClickAccent)

	if len(p.playCh) != 2 {
		t.Fatalf("expected queue len 2, got %d", len(p.playCh))
	}
	gotAccent := false
	for len(p.playCh) > 0 {
		l := <-p.playCh
		if l == ClickAccent {
			gotAccent = true
		}
	}
	if !gotAccent {
		t.Fatal("accent click should remain in queue after dropping a soft")
	}
}

func TestEnqueueDropsIncomingSoftWhenFull(t *testing.T) {
	p := &Player{
		playCh: make(chan ClickLevel, 1),
		stop:   make(chan struct{}),
	}
	p.playCh <- ClickRegular

	p.enqueueClick(ClickSoft)
	if len(p.playCh) != 1 {
		t.Fatalf("expected queue unchanged, got len %d", len(p.playCh))
	}
	if <-p.playCh != ClickRegular {
		t.Fatal("regular click should be preserved")
	}
}
