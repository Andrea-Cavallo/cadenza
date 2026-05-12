package session

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

const (
	FileExtension  = ".czs"
	CurrentVersion = uint32(1)
)

type SaveReason uint8

const (
	SaveReasonManual   SaveReason = iota
	SaveReasonAuto
	SaveReasonEvict
	SaveReasonShutdown
)

func (r SaveReason) String() string {
	switch r {
	case SaveReasonManual:
		return "manual"
	case SaveReasonAuto:
		return "auto"
	case SaveReasonEvict:
		return "evict"
	case SaveReasonShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Note struct {
	MIDI     int `json:"midi"`
	Tick     int `json:"tick"`
	Duration int `json:"duration"`
	Velocity int `json:"velocity"`
}

type Pattern struct {
	ID        string    `json:"id"`
	StepIndex int       `json:"step_index"`
	Notes     []Note    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type StoredChord struct {
	Root    string `json:"root"`
	Quality string `json:"quality"`
	Bars    [2]int `json:"bars"`
}

type StoredProgression struct {
	Key    string        `json:"key"`
	Mode   string        `json:"mode"`
	Chords []StoredChord `json:"chords"`
}

type MusicState struct {
	Key            string              `json:"key"`
	Scale          string              `json:"scale"`
	BPM            int                 `json:"bpm"`
	TimeSignature  string              `json:"time_signature"`
	Patterns       []Pattern           `json:"patterns"`
	HarmonyHistory []StoredProgression `json:"harmony_history"`
	EvolutionStep  int                 `json:"evolution_step"`
}

type SessionState struct {
	Version    uint32     `json:"version"`
	SessionID  string     `json:"session_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	SaveReason SaveReason `json:"save_reason"`
	Messages   []Message  `json:"messages"`
	MusicState MusicState `json:"music_state"`
}

type SessionMeta struct {
	SessionID    string
	FilePath     string
	SaveReason   SaveReason
	MessageCount int
	PatternCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SizeBytes    int64
}

type SessionConfig struct {
	Dir                string
	MaxSizeMB          int
	CheckpointInterval int
	MaxSessions        int
}

func DefaultConfig() SessionConfig {
	home, _ := os.UserHomeDir()
	return SessionConfig{
		Dir:                filepath.Join(home, ".cadenza", "sessions"),
		MaxSizeMB:          512,
		CheckpointInterval: 3,
		MaxSessions:        20,
	}
}

type SessionStore interface {
	Save(ctx context.Context, state *SessionState, reason SaveReason) error
	Load(ctx context.Context, sessionID string) (*SessionState, error)
	LoadByMessageHash(ctx context.Context, hash string) (*SessionState, error)
	List(ctx context.Context) ([]SessionMeta, error)
	Delete(ctx context.Context, sessionID string) error
	Evict(ctx context.Context, maxSizeMB int) error
}
