package audio

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

//go:embed sounds/*.wav
var soundFS embed.FS

const sampleRate = 44100

const (
	playQueueCap   = 16
	releaseQueueCap = 32
)

type ClickLevel int

const (
	ClickAccent ClickLevel = iota
	ClickRegular
	ClickSoft
)

type releaseJob struct {
	player *oto.Player
	pcmLen int
}

type Player struct {
	mu       sync.Mutex
	ctx      *oto.Context
	sets     map[string]map[ClickLevel][]byte
	scaled   map[string]map[ClickLevel][]byte
	volume   float64
	muted    bool
	soundSet string

	bufferLatency time.Duration

	playCh    chan ClickLevel
	releaseCh chan releaseJob

	activeMu sync.Mutex
	active   []*oto.Player

	stop     chan struct{}
	stopOnce sync.Once

	driverErr atomic.Bool
}

var SoundSets = []string{
	"woodblock",
	"clave",
	"rim",
	"beep",
	"clicktrack",
	"soft",
	"digital",
	"cowbell",
	"hihat",
	"stick",
	"pulse",
	"metronome",
	"glass",
}

func New() (*Player, error) {
	bufferSize := 50 * time.Millisecond
	op := &oto.NewContextOptions{
		SampleRate:      sampleRate,
		ChannelCount:    1,
		Format:            oto.FormatSignedInt16LE,
		BufferSize:        bufferSize,
		ApplicationName:   "metro",
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("oto context: %w", err)
	}
	<-ready

	p := &Player{
		ctx:           ctx,
		sets:          make(map[string]map[ClickLevel][]byte),
		scaled:        make(map[string]map[ClickLevel][]byte),
		volume:        0.8,
		soundSet:      "woodblock",
		bufferLatency: bufferSize + 15*time.Millisecond,
		playCh:        make(chan ClickLevel, playQueueCap),
		releaseCh:     make(chan releaseJob, releaseQueueCap),
		stop:          make(chan struct{}),
	}
	for _, name := range SoundSets {
		m, err := loadSet(name)
		if err != nil {
			return nil, err
		}
		p.sets[name] = m
	}
	p.rebuildScaledCacheLocked()

	go p.playWorker()
	go p.releaseWorker()
	return p, nil
}

func loadSet(name string) (map[ClickLevel][]byte, error) {
	levels := map[ClickLevel]string{
		ClickAccent:  name + "_accent.wav",
		ClickRegular: name + "_regular.wav",
		ClickSoft:    name + "_soft.wav",
	}
	out := make(map[ClickLevel][]byte, 3)
	for lvl, file := range levels {
		data, err := soundFS.ReadFile("sounds/" + file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		pcm, err := decodeWAVPCM(data)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", file, err)
		}
		out[lvl] = pcm
	}
	return out, nil
}

func decodeWAVPCM(data []byte) ([]byte, error) {
	if len(data) < 44 {
		return nil, fmt.Errorf("wav too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a wav")
	}
	i := 12
	for i+8 <= len(data) {
		chunk := string(data[i : i+4])
		size := int(binary.LittleEndian.Uint32(data[i+4 : i+8]))
		i += 8
		if chunk == "data" {
			if i+size > len(data) {
				size = len(data) - i
			}
			pcm := make([]byte, size)
			copy(pcm, data[i:i+size])
			return pcm, nil
		}
		i += size
	}
	return nil, fmt.Errorf("no data chunk")
}

func (p *Player) rebuildScaledCacheLocked() {
	vol := p.volume
	p.scaled = make(map[string]map[ClickLevel][]byte, len(p.sets))
	for name, levels := range p.sets {
		m := make(map[ClickLevel][]byte, len(levels))
		for lvl, pcm := range levels {
			m[lvl] = applyGain(pcm, vol)
		}
		p.scaled[name] = m
	}
}

func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	if v == p.volume {
		return
	}
	p.volume = v
	p.rebuildScaledCacheLocked()
}

func (p *Player) SetMuted(m bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.muted = m
}

func (p *Player) SetSoundSet(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sets[name]; ok {
		p.soundSet = name
	}
}

func (p *Player) SoundSet() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.soundSet
}

func (p *Player) NextSoundSet() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, s := range SoundSets {
		if s == p.soundSet {
			p.soundSet = SoundSets[(i+1)%len(SoundSets)]
			return p.soundSet
		}
	}
	p.soundSet = SoundSets[0]
	return p.soundSet
}

func (p *Player) PrevSoundSet() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, s := range SoundSets {
		if s == p.soundSet {
			p.soundSet = SoundSets[(i-1+len(SoundSets))%len(SoundSets)]
			return p.soundSet
		}
	}
	p.soundSet = SoundSets[0]
	return p.soundSet
}

// Play enqueues a click without blocking the metronome scheduler.
func (p *Player) Play(level ClickLevel) {
	p.enqueueClick(level)
}

func (p *Player) enqueueClick(level ClickLevel) {
	select {
	case <-p.stop:
		return
	default:
	}
	select {
	case p.playCh <- level:
		return
	default:
	}
	if level == ClickSoft {
		return
	}
	for {
		select {
		case old := <-p.playCh:
			if old == ClickSoft {
				select {
				case p.playCh <- level:
					return
				default:
					continue
				}
			}
			select {
			case p.playCh <- old:
			default:
			}
		default:
			select {
			case p.playCh <- level:
			default:
			}
			return
		}
	}
}

func (p *Player) playWorker() {
	for {
		select {
		case <-p.stop:
			return
		case level := <-p.playCh:
			p.playClick(level)
		}
	}
}

func (p *Player) playClick(level ClickLevel) {
	p.mu.Lock()
	muted := p.muted
	vol := p.volume
	set := p.soundSet
	scaled := p.scaled[set][level]
	p.mu.Unlock()

	if muted || vol <= 0 || len(scaled) == 0 {
		return
	}
	if err := p.ctx.Err(); err != nil {
		p.reportDriverErr(err)
		return
	}

	op := p.ctx.NewPlayer(bytes.NewReader(scaled))
	op.Play()

	p.trackActive(op)

	select {
	case p.releaseCh <- releaseJob{player: op, pcmLen: len(scaled)}:
	default:
		go p.releaseWhenDone(op, len(scaled))
	}
}

func (p *Player) trackActive(op *oto.Player) {
	p.activeMu.Lock()
	p.active = append(p.active, op)
	p.activeMu.Unlock()
}

func (p *Player) untrackActive(op *oto.Player) {
	p.activeMu.Lock()
	defer p.activeMu.Unlock()
	for i, a := range p.active {
		if a == op {
			p.active = append(p.active[:i], p.active[i+1:]...)
			return
		}
	}
}

func (p *Player) releaseWorker() {
	for {
		select {
		case <-p.stop:
			return
		case job := <-p.releaseCh:
			p.releaseWhenDone(job.player, job.pcmLen)
		}
	}
}

func (p *Player) releaseWhenDone(op *oto.Player, pcmLen int) {
	defer p.untrackActive(op)

	sampleDur := time.Duration(pcmLen/2) * time.Second / sampleRate
	deadline := time.Now().Add(sampleDur + p.bufferLatency + 500*time.Millisecond)

	for time.Now().Before(deadline) {
		if op.BufferedSize() == 0 && !op.IsPlaying() {
			time.Sleep(p.bufferLatency)
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	_ = op.Close()
}

func (p *Player) reportDriverErr(err error) {
	if p.driverErr.CompareAndSwap(false, true) {
		fmt.Fprintf(os.Stderr, "metro: audio driver error: %v\n", err)
	}
}

func (p *Player) Close() {
	p.stopOnce.Do(func() {
		close(p.stop)
	})
}

func applyGain(pcm []byte, vol float64) []byte {
	if vol <= 0 {
		return nil
	}
	if vol >= 1 {
		out := make([]byte, len(pcm))
		copy(out, pcm)
		return out
	}
	out := make([]byte, len(pcm))
	r := bytes.NewReader(pcm)
	w := bytes.NewBuffer(out[:0])
	for {
		var s int16
		if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
			if err == io.EOF {
				break
			}
			return pcm
		}
		v := float64(s) * vol
		if v > 32767 {
			v = 32767
		}
		if v < -32768 {
			v = -32768
		}
		_ = binary.Write(w, binary.LittleEndian, int16(v))
	}
	return w.Bytes()
}
