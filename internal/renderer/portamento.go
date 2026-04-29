package renderer

import (
	"github.com/Andrea-Cavallo/cadenza/internal/midi"
	"github.com/Andrea-Cavallo/cadenza/internal/renderer/styleprofile"
)

const (
	ccPortamentoTime = uint8(5)
	ccPortamentoOn   = uint8(65)
)

func portamentoTimeEvent(profile *styleprofile.StyleProfile) midi.MIDIEvent {
	return midi.MIDIEvent{
		Type:       midi.ControlChange,
		Tick:       0,
		Channel:    0,
		Controller: ccPortamentoTime,
		Value:      uint8(profile.Portamento.CC5Value),
	}
}

// portamentoEvents returns a CC65 event 5 ticks before the note tick.
// slide=true → CC65=127 (portamento on), slide=false → CC65=0 (off).
func portamentoEvents(tick int64, slide bool, profile *styleprofile.StyleProfile) []midi.MIDIEvent {
	if !profile.Portamento.UseCC65 {
		return nil
	}
	if tick < 10 {
		return nil
	}

	preTick := tick - 5

	value := uint8(0)
	if slide {
		value = 127
	}

	return []midi.MIDIEvent{{
		Type:       midi.ControlChange,
		Tick:       preTick,
		Channel:    0,
		Controller: ccPortamentoOn,
		Value:      value,
	}}
}
