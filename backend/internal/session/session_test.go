package session_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/session"
)

func TestSaveReasonString(t *testing.T) {
	cases := []struct {
		r    session.SaveReason
		want string
	}{
		{session.SaveReasonManual, "manual"},
		{session.SaveReasonAuto, "auto"},
		{session.SaveReasonEvict, "evict"},
		{session.SaveReasonShutdown, "shutdown"},
		{session.SaveReason(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("SaveReason(%d).String() = %q, want %q", tc.r, got, tc.want)
		}
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	original := session.ExportedHeader{
		SaveReason:   uint8(session.SaveReasonAuto),
		MessageCount: 42,
		PatternCount: 7,
		CreatedAt:    1715000000,
		UpdatedAt:    1715001000,
	}
	encoded := session.EncodeHeader(original)
	got, err := session.DecodeHeader(encoded)
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if got != original {
		t.Errorf("round-trip failed: got %+v, want %+v", got, original)
	}
}

func TestDecodeHeader_InvalidMagic(t *testing.T) {
	var buf [32]byte
	buf[0], buf[1], buf[2] = 'X', 'X', 'X'
	if _, err := session.DecodeHeader(buf); err == nil {
		t.Error("expected error for invalid magic, got nil")
	}
}

func TestDecodeHeader_InvalidVersion(t *testing.T) {
	var buf [32]byte
	buf[0], buf[1], buf[2] = 'C', 'Z', 'S'
	buf[3] = 99
	if _, err := session.DecodeHeader(buf); err == nil {
		t.Error("expected error for invalid version, got nil")
	}
}

func TestMessageHash_Deterministic(t *testing.T) {
	msgs := []session.Message{
		{Role: "user", Content: "bpm 128 key Am"},
		{Role: "assistant", Content: "pattern generated"},
	}
	h1 := session.MessageHash(msgs)
	h2 := session.MessageHash(msgs)
	if h1 != h2 {
		t.Errorf("hash non deterministico: %q != %q", h1, h2)
	}
	if len(h1) != 40 {
		t.Errorf("lunghezza hash SHA1 attesa 40, ottenuta %d", len(h1))
	}
}

func TestMessageHash_Empty(t *testing.T) {
	h := session.MessageHash(nil)
	if len(h) != 40 {
		t.Errorf("hash atteso per lista vuota, len=%d", len(h))
	}
}

func TestMessageHash_DifferentMessages(t *testing.T) {
	msgs1 := []session.Message{{Role: "user", Content: "A"}}
	msgs2 := []session.Message{{Role: "user", Content: "B"}}
	if session.MessageHash(msgs1) == session.MessageHash(msgs2) {
		t.Error("hash identico per messaggi diversi")
	}
}

func newTempStore(t *testing.T) (store *session.FileSessionStore, dir string) {
	t.Helper()
	dir = t.TempDir()
	cfg := session.SessionConfig{Dir: dir, MaxSizeMB: 100, CheckpointInterval: 3, MaxSessions: 20}
	store = session.NewFileStore(cfg)
	return
}

func sampleState(content string) *session.SessionState {
	return &session.SessionState{
		SessionID: content,
		Messages: []session.Message{
			{Role: "user", Content: content},
		},
		MusicState: session.MusicState{
			Key:           "Am",
			BPM:           128,
			TimeSignature: "4/4",
			EvolutionStep: 1,
		},
	}
}

func TestFileSessionStore_SaveCreatesFile(t *testing.T) {
	store, dir := newTempStore(t)
	state := sampleState("sessione-1")

	if err := store.Save(context.Background(), state, session.SaveReasonManual); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	czs := 0
	for _, e := range entries {
		if len(e.Name()) > 4 && e.Name()[len(e.Name())-4:] == ".czs" {
			czs++
		}
	}
	if czs != 1 {
		t.Errorf("atteso 1 file .czs, trovati %d", czs)
	}
}

func TestFileSessionStore_LoadRoundTrip(t *testing.T) {
	store, _ := newTempStore(t)
	state := sampleState("sessione-2")
	state.MusicState.EvolutionStep = 5

	if err := store.Save(context.Background(), state, session.SaveReasonAuto); err != nil {
		t.Fatalf("Save: %v", err)
	}

	hash := session.MessageHash(state.Messages)
	loaded, err := store.LoadByMessageHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("LoadByMessageHash: %v", err)
	}
	if loaded.MusicState.EvolutionStep != 5 {
		t.Errorf("EvolutionStep: got %d, want 5", loaded.MusicState.EvolutionStep)
	}
	if loaded.MusicState.Key != "Am" {
		t.Errorf("Key: got %q, want Am", loaded.MusicState.Key)
	}
	if loaded.SaveReason != session.SaveReasonAuto {
		t.Errorf("SaveReason: got %v, want Auto", loaded.SaveReason)
	}
}

func TestFileSessionStore_LoadBySessionID(t *testing.T) {
	store, _ := newTempStore(t)
	state := sampleState("sessione-3")

	if err := store.Save(context.Background(), state, session.SaveReasonManual); err != nil {
		t.Fatalf("Save: %v", err)
	}

	hash := session.MessageHash(state.Messages)
	loaded, err := store.Load(context.Background(), hash)
	if err != nil {
		t.Fatalf("Load by sessionID: %v", err)
	}
	if loaded.MusicState.Key != "Am" {
		t.Errorf("Key mismatch dopo Load")
	}
}

func TestFileSessionStore_SaveSetsTimestamps(t *testing.T) {
	store, _ := newTempStore(t)
	before := time.Now()
	state := sampleState("sessione-4")

	if err := store.Save(context.Background(), state, session.SaveReasonAuto); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := time.Now()

	if state.UpdatedAt.Before(before) || state.UpdatedAt.After(after) {
		t.Errorf("UpdatedAt fuori range: %v", state.UpdatedAt)
	}
	if state.CreatedAt.IsZero() {
		t.Error("CreatedAt non impostato")
	}
}

func TestFileSessionStore_List_SortedByUpdatedAt(t *testing.T) {
	store, _ := newTempStore(t)
	ctx := context.Background()

	stateA := &session.SessionState{
		Messages:   []session.Message{{Role: "user", Content: "sessione A unica"}},
		MusicState: session.MusicState{Key: "Am"},
	}
	_ = store.Save(ctx, stateA, session.SaveReasonAuto)

	stateB := &session.SessionState{
		Messages:   []session.Message{{Role: "user", Content: "sessione B unica"}},
		MusicState: session.MusicState{Key: "Cm"},
	}
	_ = store.Save(ctx, stateB, session.SaveReasonAuto)

	metas, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("attese 2 sessioni, trovate %d", len(metas))
	}
	// La lista è ordinata per UpdatedAt discendente: B (salvata per ultima) deve essere prima
	if !metas[0].UpdatedAt.After(metas[1].UpdatedAt) && !metas[0].UpdatedAt.Equal(metas[1].UpdatedAt) {
		t.Errorf("lista non ordinata per UpdatedAt desc: %v >= %v", metas[0].UpdatedAt, metas[1].UpdatedAt)
	}
}

func TestFileSessionStore_Delete(t *testing.T) {
	store, dir := newTempStore(t)
	ctx := context.Background()
	state := sampleState("delete-test")

	_ = store.Save(ctx, state, session.SaveReasonManual)
	hash := session.MessageHash(state.Messages)

	if err := store.Delete(ctx, hash); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	path := filepath.Join(dir, hash+".czs")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file not removed after Delete")
	}
}

func TestFileSessionStore_Evict_ByCount(t *testing.T) {
	cfg := session.SessionConfig{
		Dir:         t.TempDir(),
		MaxSizeMB:   100,
		MaxSessions: 2,
	}
	store := session.NewFileStore(cfg)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		st := &session.SessionState{
			Messages: []session.Message{
				{Role: "user", Content: fmt.Sprintf("sessione evict %d", i)},
			},
		}
		_ = store.Save(ctx, st, session.SaveReasonAuto)
	}

	if err := store.Evict(ctx, 100); err != nil {
		t.Fatalf("Evict: %v", err)
	}

	metas, _ := store.List(ctx)
	if len(metas) > 2 {
		t.Errorf("dopo evict attese ≤2 sessioni, trovate %d", len(metas))
	}
}

func TestFileSessionStore_List_EmptyDir(t *testing.T) {
	cfg := session.SessionConfig{
		Dir:         filepath.Join(t.TempDir(), "non-esiste"),
		MaxSizeMB:   100,
		MaxSessions: 20,
	}
	store := session.NewFileStore(cfg)
	metas, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List su dir non esistente dovrebbe restituire nil error, got: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("attesa lista vuota, got %d elementi", len(metas))
	}
}
