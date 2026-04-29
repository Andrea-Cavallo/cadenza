package midi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile_CreatesValidMIDI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.mid")

	events := []MIDIEvent{
		{Type: NoteOn, Tick: 0, Channel: 0, Note: 45, Velocity: 100},
		{Type: NoteOff, Tick: 480, Channel: 0, Note: 45},
		{Type: NoteOn, Tick: 480, Channel: 0, Note: 48, Velocity: 90},
		{Type: NoteOff, Tick: 960, Channel: 0, Note: 48},
	}

	w := NewWriter(120.0)
	err := w.WriteFile(path, events)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("file is empty")
	}

	// Verify MIDI header magic bytes
	data, _ := os.ReadFile(path)
	if string(data[0:4]) != "MThd" {
		t.Errorf("expected MThd header, got %q", string(data[0:4]))
	}
	if string(data[14:18]) != "MTrk" {
		t.Errorf("expected MTrk chunk, got %q", string(data[14:18]))
	}
}

func TestWriteFile_WithCCEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_cc.mid")

	events := []MIDIEvent{
		{Type: ControlChange, Tick: 0, Channel: 0, Controller: 74, Value: 64},
		{Type: NoteOn, Tick: 0, Channel: 0, Note: 45, Velocity: 100},
		{Type: NoteOff, Tick: 480, Channel: 0, Note: 45},
	}

	w := NewWriter(122.0)
	err := w.WriteFile(path, events)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}

func TestVarLen(t *testing.T) {
	tests := []struct {
		val  int64
		want []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7F}},
		{128, []byte{0x81, 0x00}},
		{480, []byte{0x83, 0x60}},
	}
	for _, tt := range tests {
		got := varLen(tt.val)
		if len(got) != len(tt.want) {
			t.Errorf("varLen(%d) len=%d want len=%d", tt.val, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("varLen(%d)[%d] = 0x%02X, want 0x%02X", tt.val, i, got[i], tt.want[i])
			}
		}
	}
}
