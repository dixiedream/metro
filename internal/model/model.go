package model

import "fmt"

const (
	MinBPM = 10.0
	MaxBPM = 400.0
)

type BeatKind int

const (
	BeatSilent BeatKind = iota
	BeatRegular
	BeatAccent
)

func (b BeatKind) String() string {
	switch b {
	case BeatSilent:
		return "silent"
	case BeatAccent:
		return "accent"
	default:
		return "regular"
	}
}

func (b BeatKind) Cycle() BeatKind {
	switch b {
	case BeatAccent:
		return BeatRegular
	case BeatRegular:
		return BeatSilent
	default:
		return BeatAccent
	}
}

type Mode int

const (
	ModeBasic Mode = iota
	ModePreset
	ModePractice
)

func (m Mode) String() string {
	switch m {
	case ModeBasic:
		return "Basic"
	case ModePreset:
		return "Preset"
	case ModePractice:
		return "Practice"
	default:
		return "Unknown"
	}
}

func AllModes() []Mode {
	return []Mode{ModeBasic, ModePreset, ModePractice}
}

// NormalizeMode maps legacy saved mode values (from when Setlist/Gig existed) to current modes.
func NormalizeMode(m Mode) Mode {
	switch int(m) {
	case 0:
		return ModeBasic
	case 1:
		return ModePreset
	case 2, 4:
		return ModeBasic
	case 3:
		return ModePractice
	default:
		return ModeBasic
	}
}

type TrackerMode int

const (
	TrackerOff TrackerMode = iota
	TrackerBars
	TrackerTime
)

func (t TrackerMode) String() string {
	switch t {
	case TrackerBars:
		return "bars"
	case TrackerTime:
		return "time"
	default:
		return "off"
	}
}

type Unit int

const (
	UnitBars Unit = iota
	UnitSeconds
)

func (u Unit) String() string {
	if u == UnitSeconds {
		return "seconds"
	}
	return "bars"
}

type Config struct {
	BPM         float64    `json:"bpm"`
	Numerator   int        `json:"numerator"`
	Denominator int        `json:"denominator"`
	PatternID   string     `json:"pattern_id"`
	Beats       []BeatKind `json:"beats"`
	SoundSet    string     `json:"sound_set"`
	Volume      float64    `json:"volume"`
	Muted       bool       `json:"muted"`
}

func DefaultConfig() Config {
	c := Config{
		BPM:         120,
		Numerator:   4,
		Denominator: 4,
		PatternID:   "quarter",
		SoundSet:    "woodblock",
		Volume:      0.8,
		Muted:       false,
	}
	c.Beats = DefaultBeats(c.Numerator)
	return c
}

func DefaultBeats(n int) []BeatKind {
	if n < 1 {
		n = 1
	}
	beats := make([]BeatKind, n)
	for i := range beats {
		if i == 0 {
			beats[i] = BeatAccent
		} else {
			beats[i] = BeatRegular
		}
	}
	return beats
}

func (c *Config) Clamp() {
	if c.BPM < MinBPM {
		c.BPM = MinBPM
	}
	if c.BPM > MaxBPM {
		c.BPM = MaxBPM
	}
	if c.Numerator < 1 {
		c.Numerator = 1
	}
	if c.Numerator > 13 {
		c.Numerator = 13
	}
	switch c.Denominator {
	case 2, 4, 8:
	default:
		c.Denominator = 4
	}
	if c.Volume < 0 {
		c.Volume = 0
	}
	if c.Volume > 1 {
		c.Volume = 1
	}
	if len(c.Beats) != c.Numerator {
		c.Beats = DefaultBeats(c.Numerator)
	}
	if c.PatternID == "" {
		c.PatternID = "quarter"
	}
	if c.SoundSet == "" {
		c.SoundSet = "woodblock"
	}
}

func (c Config) Signature() string {
	return fmt.Sprintf("%d/%d", c.Numerator, c.Denominator)
}

func (c Config) IsCompound() bool {
	return c.Denominator == 8 && (c.Numerator == 3 || c.Numerator == 6 || c.Numerator == 9 || c.Numerator == 12)
}

type Tracker struct {
	Mode     TrackerMode `json:"mode"`
	Limit    int         `json:"limit"`
	Elapsed  int         `json:"-"`
	AutoStop bool        `json:"auto_stop"`
}

func DefaultTracker() Tracker {
	return Tracker{Mode: TrackerOff, Limit: 16, AutoStop: true}
}

type Automator struct {
	Enabled  bool    `json:"enabled"`
	DeltaBPM float64 `json:"delta_bpm"`
	EveryN   int     `json:"every_n"`
	Unit     Unit    `json:"unit"`
}

func DefaultAutomator() Automator {
	return Automator{Enabled: false, DeltaBPM: 5, EveryN: 4, Unit: UnitBars}
}

type Coach struct {
	Enabled    bool `json:"enabled"`
	PlayBars   int  `json:"play_bars"`
	SilentBars int  `json:"silent_bars"`
}

func DefaultCoach() Coach {
	return Coach{Enabled: false, PlayBars: 2, SilentBars: 2}
}

type MIDIBindings struct {
	Enabled    bool   `json:"enabled"`
	DeviceName string `json:"device_name"`
	PlayPause  int    `json:"play_pause"` // note number, -1 = unused
	TempoUp    int    `json:"tempo_up"`
	TempoDown  int    `json:"tempo_down"`
}

func DefaultMIDIBindings() MIDIBindings {
	return MIDIBindings{
		Enabled:    false,
		DeviceName: "",
		PlayPause:  36, // C1 / kick
		TempoUp:    38,
		TempoDown:  40,
	}
}

type Settings struct {
	Mode      Mode         `json:"mode"`
	Volume    float64      `json:"volume"`
	MIDI      MIDIBindings `json:"midi"`
	Config    Config       `json:"config"`
	Tracker   Tracker      `json:"tracker"`
	Automator Automator    `json:"automator"`
	Coach     Coach        `json:"coach"`
}

func DefaultSettings() Settings {
	return Settings{
		Mode:      ModeBasic,
		Volume:    0.8,
		MIDI:      DefaultMIDIBindings(),
		Config:    DefaultConfig(),
		Tracker:   DefaultTracker(),
		Automator: DefaultAutomator(),
		Coach:     DefaultCoach(),
	}
}

// TempoMarking returns the classical/Italian tempo marking for a BPM.
func TempoMarking(bpm float64) string {
	switch {
	case bpm < 20:
		return "Larghissimo"
	case bpm < 40:
		return "Grave"
	case bpm < 45:
		return "Largo"
	case bpm < 50:
		return "Lento"
	case bpm < 55:
		return "Larghetto"
	case bpm < 65:
		return "Adagio"
	case bpm < 69:
		return "Adagietto"
	case bpm < 73:
		return "Andante"
	case bpm < 78:
		return "Andantino"
	case bpm < 83:
		return "Marcia moderato"
	case bpm < 86:
		return "Andante moderato"
	case bpm < 98:
		return "Moderato"
	case bpm < 110:
		return "Allegretto"
	case bpm < 132:
		return "Allegro"
	case bpm < 140:
		return "Vivace"
	case bpm < 168:
		return "Vivacissimo"
	case bpm < 177:
		return "Allegrissimo"
	case bpm < 200:
		return "Presto"
	default:
		return "Prestissimo"
	}
}

// ValidSignatures lists Android Tempo-compatible signatures.
func ValidSignatures() [][2]int {
	out := make([][2]int, 0, 30)
	for n := 1; n <= 13; n++ {
		out = append(out, [2]int{n, 2})
	}
	for n := 1; n <= 13; n++ {
		out = append(out, [2]int{n, 4})
	}
	for _, n := range []int{3, 5, 6, 7, 9, 12} {
		out = append(out, [2]int{n, 8})
	}
	return out
}

func NextSignature(num, den int) (int, int) {
	sigs := ValidSignatures()
	for i, s := range sigs {
		if s[0] == num && s[1] == den {
			n := sigs[(i+1)%len(sigs)]
			return n[0], n[1]
		}
	}
	return 4, 4
}

func PrevSignature(num, den int) (int, int) {
	sigs := ValidSignatures()
	for i, s := range sigs {
		if s[0] == num && s[1] == den {
			n := sigs[(i-1+len(sigs))%len(sigs)]
			return n[0], n[1]
		}
	}
	return 4, 4
}
