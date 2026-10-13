package midi

import (
	"fmt"
	"sync"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"

	"github.com/ale/metro/internal/model"
)

type Action int

const (
	ActionNone Action = iota
	ActionPlayPause
	ActionTempoUp
	ActionTempoDown
)

type Event struct {
	Action Action
}

type Listener struct {
	mu       sync.Mutex
	bindings model.MIDIBindings
	stop     func()
	events   chan Event
}

func New() *Listener {
	return &Listener{events: make(chan Event, 16)}
}

func (l *Listener) Events() <-chan Event { return l.events }

func (l *Listener) Configure(b model.MIDIBindings) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopLocked()
	l.bindings = b
	if !b.Enabled {
		return nil
	}
	ins := midi.GetInPorts()
	if len(ins) == 0 {
		return fmt.Errorf("no MIDI input ports")
	}
	var in drivers.In
	if b.DeviceName != "" {
		for _, p := range ins {
			if p.String() == b.DeviceName {
				in = p
				break
			}
		}
		if in == nil {
			return fmt.Errorf("MIDI device %q not found", b.DeviceName)
		}
	} else {
		in = ins[0]
	}
	stop, err := midi.ListenTo(in, func(msg midi.Message, _ int32) {
		var ch, key, vel uint8
		if !msg.GetNoteStart(&ch, &key, &vel) {
			return
		}
		l.mu.Lock()
		b := l.bindings
		l.mu.Unlock()
		act := ActionNone
		switch int(key) {
		case b.PlayPause:
			act = ActionPlayPause
		case b.TempoUp:
			act = ActionTempoUp
		case b.TempoDown:
			act = ActionTempoDown
		}
		if act != ActionNone {
			select {
			case l.events <- Event{Action: act}:
			default:
			}
		}
	})
	if err != nil {
		return err
	}
	l.stop = stop
	return nil
}

func (l *Listener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stopLocked()
	midi.CloseDriver()
}

func (l *Listener) stopLocked() {
	if l.stop != nil {
		l.stop()
		l.stop = nil
	}
}

func ListDevices() []string {
	ins := midi.GetInPorts()
	out := make([]string, 0, len(ins))
	for _, p := range ins {
		out = append(out, p.String())
	}
	return out
}
