package midi

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	ticksPerBeat = 480
	channel      = 0
)

type EventType int

const (
	NoteOn EventType = iota
	NoteOff
	ControlChange
)

type MIDIEvent struct {
	Type       EventType
	Tick       int64
	Channel    uint8
	Note       uint8
	Velocity   uint8
	Controller uint8
	Value      uint8
}

type Writer struct {
	bpm float64
}

func NewWriter(bpm float64) *Writer {
	return &Writer{bpm: bpm}
}

func (w *Writer) WriteFile(path string, events []MIDIEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	sorted := make([]MIDIEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Tick != sorted[j].Tick {
			return sorted[i].Tick < sorted[j].Tick
		}
		return eventPriority(sorted[i].Type) < eventPriority(sorted[j].Type)
	})

	trackData := w.buildTrack(sorted)
	header := buildHeader(len(trackData))

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := f.Write(trackData); err != nil {
		return fmt.Errorf("write track: %w", err)
	}
	return nil
}

func buildHeader(trackLen int) []byte {
	h := make([]byte, 14)
	copy(h[0:4], "MThd")
	binary.BigEndian.PutUint32(h[4:8], 6) // header length
	binary.BigEndian.PutUint16(h[8:10], 0) // format 0
	binary.BigEndian.PutUint16(h[10:12], 1) // 1 track
	binary.BigEndian.PutUint16(h[12:14], ticksPerBeat)
	return h
}

func (w *Writer) buildTrack(events []MIDIEvent) []byte {
	var body []byte

	// tempo event
	microsecondsPerBeat := uint32(60_000_000 / w.bpm)
	body = append(body, varLen(0)...)         // delta 0
	body = append(body, 0xFF, 0x51, 0x03)     // meta tempo
	body = append(body,
		byte(microsecondsPerBeat>>16),
		byte(microsecondsPerBeat>>8),
		byte(microsecondsPerBeat),
	)

	// time signature 4/4
	body = append(body, varLen(0)...)
	body = append(body, 0xFF, 0x58, 0x04, 4, 2, 24, 8)

	var lastTick int64
	for _, ev := range events {
		delta := ev.Tick - lastTick
		lastTick = ev.Tick
		body = append(body, varLen(delta)...)

		switch ev.Type {
		case NoteOn:
			body = append(body, 0x90|(ev.Channel&0x0F), ev.Note&0x7F, ev.Velocity&0x7F)
		case NoteOff:
			body = append(body, 0x80|(ev.Channel&0x0F), ev.Note&0x7F, 0)
		case ControlChange:
			body = append(body, 0xB0|(ev.Channel&0x0F), ev.Controller&0x7F, ev.Value&0x7F)
		}
	}

	// end of track
	body = append(body, varLen(0)...)
	body = append(body, 0xFF, 0x2F, 0x00)

	chunk := make([]byte, 8+len(body))
	copy(chunk[0:4], "MTrk")
	binary.BigEndian.PutUint32(chunk[4:8], uint32(len(body)))
	copy(chunk[8:], body)
	return chunk
}

func eventPriority(t EventType) int {
	switch t {
	case ControlChange:
		return 0
	case NoteOff:
		return 1
	case NoteOn:
		return 2
	default:
		return 3
	}
}

// varLen encodes a value as MIDI variable-length quantity.
func varLen(v int64) []byte {
	if v < 0 {
		v = 0
	}
	if v < 0x80 {
		return []byte{byte(v)}
	}
	var buf [4]byte
	n := 0
	for v > 0 {
		buf[n] = byte(v & 0x7F)
		v >>= 7
		n++
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[n-1-i] = buf[i]
		if i > 0 {
			out[n-1-i] |= 0x80
		}
	}
	return out
}
