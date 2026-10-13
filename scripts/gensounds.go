package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
)

// Generates short 16-bit mono 44100 Hz WAV click samples for metro sound sets.
func main() {
	outDir := "internal/audio/sounds"
	_ = os.MkdirAll(outDir, 0o755)

	sets := []struct {
		name string
		freq float64
		dur  float64
		kind string
	}{
		{"woodblock_accent", 980, 0.035, "click"},
		{"woodblock_regular", 720, 0.028, "click"},
		{"woodblock_soft", 540, 0.022, "click"},
		{"clave_accent", 1400, 0.03, "click"},
		{"clave_regular", 1100, 0.025, "click"},
		{"clave_soft", 850, 0.02, "click"},
		{"rim_accent", 400, 0.04, "noise"},
		{"rim_regular", 320, 0.032, "noise"},
		{"rim_soft", 260, 0.025, "noise"},
		{"beep_accent", 1200, 0.04, "tone"},
		{"beep_regular", 880, 0.035, "tone"},
		{"beep_soft", 660, 0.028, "tone"},
		{"clicktrack_accent", 2000, 0.02, "click"},
		{"clicktrack_regular", 1600, 0.016, "click"},
		{"clicktrack_soft", 1200, 0.012, "click"},
		{"soft_accent", 600, 0.05, "tone"},
		{"soft_regular", 480, 0.045, "tone"},
		{"soft_soft", 400, 0.04, "tone"},
		{"digital_accent", 1800, 0.018, "square"},
		{"digital_regular", 1400, 0.015, "square"},
		{"digital_soft", 1000, 0.012, "square"},
		{"cowbell_accent", 800, 0.06, "metallic"},
		{"cowbell_regular", 650, 0.05, "metallic"},
		{"cowbell_soft", 520, 0.04, "metallic"},
		{"hihat_accent", 9000, 0.025, "noise"},
		{"hihat_regular", 7000, 0.02, "noise"},
		{"hihat_soft", 5000, 0.015, "noise"},
		{"stick_accent", 1100, 0.03, "click"},
		{"stick_regular", 900, 0.025, "click"},
		{"stick_soft", 700, 0.02, "click"},
		{"pulse_accent", 220, 0.05, "tone"},
		{"pulse_regular", 180, 0.045, "tone"},
		{"pulse_soft", 140, 0.04, "tone"},
		{"metronome_accent", 1320, 0.04, "click"},
		{"metronome_regular", 880, 0.035, "click"},
		{"metronome_soft", 660, 0.028, "click"},
		{"glass_accent", 2400, 0.05, "metallic"},
		{"glass_regular", 1800, 0.04, "metallic"},
		{"glass_soft", 1400, 0.03, "metallic"},
	}

	for _, s := range sets {
		samples := synthesize(s.freq, s.dur, s.kind)
		path := filepath.Join(outDir, s.name+".wav")
		if err := writeWAV(path, samples, 44100); err != nil {
			panic(err)
		}
	}
}

func synthesize(freq, dur float64, kind string) []int16 {
	sr := 44100.0
	n := int(dur * sr)
	out := make([]int16, n)
	for i := 0; i < n; i++ {
		t := float64(i) / sr
		env := math.Exp(-t * (12 + freq/200))
		var sample float64
		switch kind {
		case "tone":
			sample = math.Sin(2 * math.Pi * freq * t)
		case "square":
			if math.Sin(2*math.Pi*freq*t) >= 0 {
				sample = 0.6
			} else {
				sample = -0.6
			}
		case "noise":
			// deterministic pseudo-noise shaped by freq
			sample = math.Sin(2*math.Pi*freq*t+float64(i)*12.9898)*0.5 +
				math.Sin(2*math.Pi*(freq*1.7)*t)*0.3
		case "metallic":
			sample = math.Sin(2*math.Pi*freq*t)*0.5 +
				math.Sin(2*math.Pi*freq*2.7*t)*0.3 +
				math.Sin(2*math.Pi*freq*4.1*t)*0.2
		default: // click
			sample = math.Sin(2*math.Pi*freq*t) * (1 - t/dur)
		}
		v := sample * env * 0.85
		if v > 1 {
			v = 1
		}
		if v < -1 {
			v = -1
		}
		out[i] = int16(v * 32767)
	}
	return out
}

func writeWAV(path string, samples []int16, sampleRate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dataSize := len(samples) * 2
	var hdr [44]byte
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+dataSize))
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1) // PCM
	binary.LittleEndian.PutUint16(hdr[22:], 1) // mono
	binary.LittleEndian.PutUint32(hdr[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(hdr[28:], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(hdr[32:], 2)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(dataSize))
	if _, err := f.Write(hdr[:]); err != nil {
		return err
	}
	return binary.Write(f, binary.LittleEndian, samples)
}
