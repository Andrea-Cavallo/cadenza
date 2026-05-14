# Session Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Aggiungere un sistema di persistenza delle sessioni Cadenza su disco (formato `.czs`) che salvi stato musicale e history LLM, con CLI `--resume` / `--list-sessions` e shutdown hook.

**Architecture:** Header binario fisso 32 byte + payload JSON per ogni sessione; naming basato su SHA1 della message history per deduplicazione automatica; `FileSessionStore` implementa `SessionStore` con eviction FIFO. Il package `internal/session/` è autonomo e non importa package interni Cadenza (nessuna dipendenza circolare).

**Tech Stack:** Go 1.25, stdlib only (`encoding/binary`, `crypto/sha1`, `encoding/json`, `os`, `sync`), `github.com/google/uuid` già presente in `go.mod`.

---

## File map

| Azione | File | Responsabilità |
|--------|------|----------------|
| Crea | `internal/session/store.go` | Tipi pubblici + interfaccia `SessionStore` |
| Crea | `internal/session/header.go` | Encode/decode header binario 32 byte |
| Crea | `internal/session/hash.go` | SHA1 deterministica della message history |
| Crea | `internal/session/file_store.go` | `FileSessionStore` — impl. completa |
| Crea | `internal/session/session_test.go` | Test unitari per tutti i file sopra |
| Modifica | `internal/config/config.go` | Aggiunge `SessionSection` ad `AppConfig` |
| Modifica | `cmd/cadenza/main.go` | Flag `--resume`, `--list-sessions`, shutdown hook, checkpoint |

---

## Task 1: Tipi e interfaccia

**Files:**
- Crea: `internal/session/store.go`
- Test: `internal/session/session_test.go`

- [ ] **Step 1.1: Scrivi il test per `SaveReason.String()`**

Crea `internal/session/session_test.go`:

```go
package session_test

import (
	"testing"

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
```

- [ ] **Step 1.2: Esegui il test — deve fallire**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... 2>&1
```

Expected: `no Go files in ...session` o `cannot find package`.

- [ ] **Step 1.3: Crea `internal/session/store.go`**

```go
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

// SaveReason identifica il motivo del salvataggio.
type SaveReason uint8

const (
	SaveReasonManual   SaveReason = iota // cadenza --save
	SaveReasonAuto                       // checkpoint periodico
	SaveReasonEvict                      // sessione soppiantata
	SaveReasonShutdown                   // Ctrl+C / exit pulito
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

// Message rappresenta un messaggio nella history LLM.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Note è una nota MIDI memorizzata nella sessione.
type Note struct {
	MIDI     int `json:"midi"`
	Tick     int `json:"tick"`
	Duration int `json:"duration"`
	Velocity int `json:"velocity"`
}

// Pattern è un pattern generato durante la sessione.
type Pattern struct {
	ID        string    `json:"id"`
	StepIndex int       `json:"step_index"`
	Notes     []Note    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// StoredChord è un accordo serializzabile (evita dipendenze circolari con theory).
type StoredChord struct {
	Root    string `json:"root"`
	Quality string `json:"quality"`
	Bars    [2]int `json:"bars"`
}

// StoredProgression è una progressione armonica serializzabile.
type StoredProgression struct {
	Key    string        `json:"key"`
	Mode   string        `json:"mode"`
	Chords []StoredChord `json:"chords"`
}

// MusicState contiene lo stato musicale della sessione.
type MusicState struct {
	Key            string              `json:"key"`
	Scale          string              `json:"scale"`
	BPM            int                 `json:"bpm"`
	TimeSignature  string              `json:"time_signature"`
	Patterns       []Pattern           `json:"patterns"`
	HarmonyHistory []StoredProgression `json:"harmony_history"`
	EvolutionStep  int                 `json:"evolution_step"`
}

// SessionState è lo snapshot completo di una sessione.
type SessionState struct {
	Version    uint32     `json:"version"`
	SessionID  string     `json:"session_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	SaveReason SaveReason `json:"save_reason"`
	Messages   []Message  `json:"messages"`
	MusicState MusicState `json:"music_state"`
}

// SessionMeta contiene i metadati di una sessione (senza caricare il payload).
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

// SessionConfig parametrizza il comportamento del store.
type SessionConfig struct {
	Dir                string
	MaxSizeMB          int
	CheckpointInterval int
	MaxSessions        int
}

// DefaultConfig restituisce la configurazione di default.
func DefaultConfig() SessionConfig {
	home, _ := os.UserHomeDir()
	return SessionConfig{
		Dir:                filepath.Join(home, ".cadenza", "sessions"),
		MaxSizeMB:          512,
		CheckpointInterval: 3,
		MaxSessions:        20,
	}
}

// SessionStore definisce le operazioni sul repository delle sessioni.
type SessionStore interface {
	Save(ctx context.Context, state *SessionState, reason SaveReason) error
	Load(ctx context.Context, sessionID string) (*SessionState, error)
	LoadByMessageHash(ctx context.Context, hash string) (*SessionState, error)
	List(ctx context.Context) ([]SessionMeta, error)
	Delete(ctx context.Context, sessionID string) error
	Evict(ctx context.Context, maxSizeMB int) error
}
```

- [ ] **Step 1.4: Esegui il test — deve passare**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -v -run TestSaveReasonString
```

Expected: `PASS`.

- [ ] **Step 1.5: Commit**

```
git add internal/session/store.go internal/session/session_test.go
git commit -m "feat(session): tipi, interfaccia SessionStore e SaveReason"
```

---

## Task 2: Header binario

**Files:**
- Crea: `internal/session/header.go`
- Test: `internal/session/session_test.go` (aggiunta)

- [ ] **Step 2.1: Aggiungi i test per header round-trip a `session_test.go`**

```go
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
	buf[3] = 99 // versione non supportata
	if _, err := session.DecodeHeader(buf); err == nil {
		t.Error("expected error for invalid version, got nil")
	}
}
```

- [ ] **Step 2.2: Esegui i test — devono fallire**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -run TestHeader 2>&1
```

Expected: errori di compilazione (`ExportedHeader`, `EncodeHeader`, `DecodeHeader` non definiti).

- [ ] **Step 2.3: Crea `internal/session/header.go`**

```go
package session

import (
	"encoding/binary"
	"fmt"
)

const headerSize = 32

var headerMagic = [3]byte{'C', 'Z', 'S'}

const headerFileVersion = uint8(1)

// ExportedHeader è esportato per permettere i test di round-trip.
type ExportedHeader struct {
	SaveReason   uint8
	MessageCount uint32
	PatternCount uint32
	CreatedAt    int64
	UpdatedAt    int64
}

// EncodeHeader serializza un header in 32 byte (little-endian).
func EncodeHeader(h ExportedHeader) [headerSize]byte {
	var buf [headerSize]byte
	buf[0] = headerMagic[0]
	buf[1] = headerMagic[1]
	buf[2] = headerMagic[2]
	buf[3] = headerFileVersion
	buf[4] = h.SaveReason
	// buf[5:8] reserved = 0
	binary.LittleEndian.PutUint32(buf[8:12], h.MessageCount)
	binary.LittleEndian.PutUint32(buf[12:16], h.PatternCount)
	binary.LittleEndian.PutUint64(buf[16:24], uint64(h.CreatedAt))
	binary.LittleEndian.PutUint64(buf[24:32], uint64(h.UpdatedAt))
	return buf
}

// DecodeHeader deserializza 32 byte in un ExportedHeader.
func DecodeHeader(buf [headerSize]byte) (ExportedHeader, error) {
	if buf[0] != headerMagic[0] || buf[1] != headerMagic[1] || buf[2] != headerMagic[2] {
		return ExportedHeader{}, fmt.Errorf("magic non valido: %q", buf[0:3])
	}
	if buf[3] != headerFileVersion {
		return ExportedHeader{}, fmt.Errorf("versione non supportata: %d", buf[3])
	}
	return ExportedHeader{
		SaveReason:   buf[4],
		MessageCount: binary.LittleEndian.Uint32(buf[8:12]),
		PatternCount: binary.LittleEndian.Uint32(buf[12:16]),
		CreatedAt:    int64(binary.LittleEndian.Uint64(buf[16:24])),
		UpdatedAt:    int64(binary.LittleEndian.Uint64(buf[24:32])),
	}, nil
}
```

- [ ] **Step 2.4: Esegui i test — devono passare**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -v -run TestHeader
```

Expected: 3 test `PASS`.

- [ ] **Step 2.5: Commit**

```
git add internal/session/header.go internal/session/session_test.go
git commit -m "feat(session): header binario 32 byte con encode/decode"
```

---

## Task 3: Message hash

**Files:**
- Crea: `internal/session/hash.go`
- Test: `internal/session/session_test.go` (aggiunta)

- [ ] **Step 3.1: Aggiungi i test per `MessageHash` a `session_test.go`**

```go
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
```

- [ ] **Step 3.2: Esegui i test — devono fallire**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -run TestMessageHash 2>&1
```

Expected: `MessageHash` non definita.

- [ ] **Step 3.3: Crea `internal/session/hash.go`**

```go
package session

import (
	"crypto/sha1"
	"encoding/hex"
)

// MessageHash calcola SHA1 della sequenza di messaggi.
// Due sessioni con la stessa history LLM producono lo stesso hash.
func MessageHash(msgs []Message) string {
	h := sha1.New()
	for _, m := range msgs {
		h.Write([]byte(m.Role))
		h.Write([]byte{0})
		h.Write([]byte(m.Content))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
```

- [ ] **Step 3.4: Esegui i test — devono passare**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -v -run TestMessageHash
```

Expected: 3 test `PASS`.

- [ ] **Step 3.5: Commit**

```
git add internal/session/hash.go internal/session/session_test.go
git commit -m "feat(session): SHA1 deterministica della message history"
```

---

## Task 4: FileSessionStore — Save e Load

**Files:**
- Crea: `internal/session/file_store.go`
- Test: `internal/session/session_test.go` (aggiunta)

- [ ] **Step 4.1: Aggiungi i test per Save e Load**

```go
import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/session"
)

func newTempStore(t *testing.T) (*session.FileSessionStore, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := session.SessionConfig{Dir: dir, MaxSizeMB: 100, CheckpointInterval: 3, MaxSessions: 20}
	return session.NewFileStore(cfg), dir
}

func sampleState(sessionID string) *session.SessionState {
	return &session.SessionState{
		SessionID: sessionID,
		Messages: []session.Message{
			{Role: "user", Content: "key Am bpm 128"},
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
	state := sampleState("test-session-1")

	if err := store.Save(context.Background(), state, session.SaveReasonManual); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	czs := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".czs") {
			czs++
		}
	}
	if czs != 1 {
		t.Errorf("atteso 1 file .czs, trovati %d", czs)
	}
}

func TestFileSessionStore_LoadRoundTrip(t *testing.T) {
	store, _ := newTempStore(t)
	state := sampleState("test-session-2")
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
		t.Errorf("Key: got %q, want %q", loaded.MusicState.Key, "Am")
	}
	if loaded.SaveReason != session.SaveReasonAuto {
		t.Errorf("SaveReason: got %v, want Auto", loaded.SaveReason)
	}
}

func TestFileSessionStore_LoadBySessionID(t *testing.T) {
	store, _ := newTempStore(t)
	state := sampleState("test-session-3")

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
	state := sampleState("test-session-4")

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
```

- [ ] **Step 4.2: Esegui i test — devono fallire**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -run "TestFileSessionStore_Save|TestFileSessionStore_Load" 2>&1
```

Expected: `NewFileStore` non definito.

- [ ] **Step 4.3: Crea `internal/session/file_store.go`**

```go
package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileSessionStore salva le sessioni come file .czs su disco.
type FileSessionStore struct {
	cfg SessionConfig
	mu  sync.Mutex
}

// NewFileStore restituisce uno store pronto all'uso con la configurazione data.
func NewFileStore(cfg SessionConfig) *FileSessionStore {
	return &FileSessionStore{cfg: cfg}
}

func (s *FileSessionStore) Save(ctx context.Context, state *SessionState, reason SaveReason) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.Dir, 0o755); err != nil {
		return fmt.Errorf("crea directory sessioni: %w", err)
	}

	now := time.Now()
	state.SaveReason = reason
	state.UpdatedAt = now
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	if state.Version == 0 {
		state.Version = CurrentVersion
	}

	hash := MessageHash(state.Messages)
	path := filepath.Join(s.cfg.Dir, hash+FileExtension)

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal sessione: %w", err)
	}

	h := ExportedHeader{
		SaveReason:   uint8(reason),
		MessageCount: uint32(len(state.Messages)),
		PatternCount: uint32(len(state.MusicState.Patterns)),
		CreatedAt:    state.CreatedAt.Unix(),
		UpdatedAt:    state.UpdatedAt.Unix(),
	}
	rawHdr := EncodeHeader(h)

	var buf bytes.Buffer
	buf.Write(rawHdr[:])
	buf.Write(payload)

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("scrivi file sessione: %w", err)
	}

	slog.Info("sessione salvata",
		"path", path,
		"reason", reason.String(),
		"messages", len(state.Messages),
		"patterns", len(state.MusicState.Patterns),
	)
	return nil
}

func (s *FileSessionStore) Load(ctx context.Context, sessionID string) (*SessionState, error) {
	path := filepath.Join(s.cfg.Dir, sessionID+FileExtension)
	return s.loadFile(path)
}

func (s *FileSessionStore) LoadByMessageHash(ctx context.Context, hash string) (*SessionState, error) {
	path := filepath.Join(s.cfg.Dir, hash+FileExtension)
	return s.loadFile(path)
}

func (s *FileSessionStore) loadFile(path string) (*SessionState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leggi file sessione: %w", err)
	}
	if len(data) < headerSize {
		return nil, fmt.Errorf("file sessione corrotto: troppo corto (%d byte)", len(data))
	}

	var rawHdr [headerSize]byte
	copy(rawHdr[:], data[:headerSize])
	if _, err := DecodeHeader(rawHdr); err != nil {
		return nil, fmt.Errorf("header sessione non valido: %w", err)
	}

	var state SessionState
	if err := json.Unmarshal(data[headerSize:], &state); err != nil {
		return nil, fmt.Errorf("unmarshal sessione: %w", err)
	}
	return &state, nil
}

func (s *FileSessionStore) List(ctx context.Context) ([]SessionMeta, error) {
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("leggi directory sessioni: %w", err)
	}

	var metas []SessionMeta
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), FileExtension) {
			continue
		}
		path := filepath.Join(s.cfg.Dir, e.Name())
		meta, err := s.readMeta(path, e)
		if err != nil {
			slog.Warn("sessione ignorata", "path", path, "err", err)
			continue
		}
		metas = append(metas, meta)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.After(metas[j].UpdatedAt)
	})
	return metas, nil
}

func (s *FileSessionStore) readMeta(path string, e os.DirEntry) (SessionMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionMeta{}, fmt.Errorf("apri file: %w", err)
	}
	defer f.Close()

	var rawHdr [headerSize]byte
	if _, err := io.ReadFull(f, rawHdr[:]); err != nil {
		return SessionMeta{}, fmt.Errorf("leggi header: %w", err)
	}
	h, err := DecodeHeader(rawHdr)
	if err != nil {
		return SessionMeta{}, err
	}

	info, _ := e.Info()
	var size int64
	if info != nil {
		size = info.Size()
	}

	name := e.Name()
	sessionID := name[:len(name)-len(FileExtension)]

	return SessionMeta{
		SessionID:    sessionID,
		FilePath:     path,
		SaveReason:   SaveReason(h.SaveReason),
		MessageCount: int(h.MessageCount),
		PatternCount: int(h.PatternCount),
		CreatedAt:    time.Unix(h.CreatedAt, 0),
		UpdatedAt:    time.Unix(h.UpdatedAt, 0),
		SizeBytes:    size,
	}, nil
}

func (s *FileSessionStore) Delete(ctx context.Context, sessionID string) error {
	path := filepath.Join(s.cfg.Dir, sessionID+FileExtension)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("elimina sessione %s: %w", sessionID, err)
	}
	slog.Info("sessione eliminata", "session_id", sessionID)
	return nil
}

func (s *FileSessionStore) Evict(ctx context.Context, maxSizeMB int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metas, err := s.List(ctx)
	if err != nil {
		return err
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].UpdatedAt.Before(metas[j].UpdatedAt)
	})

	var totalBytes int64
	for _, m := range metas {
		totalBytes += m.SizeBytes
	}
	maxBytes := int64(maxSizeMB) * 1024 * 1024

	for len(metas) > s.cfg.MaxSessions || totalBytes > maxBytes {
		if len(metas) == 0 {
			break
		}
		oldest := metas[0]
		metas = metas[1:]
		totalBytes -= oldest.SizeBytes

		if err := os.Remove(oldest.FilePath); err != nil {
			slog.Warn("evict: impossibile eliminare", "path", oldest.FilePath, "err", err)
			continue
		}
		slog.Info("evict: sessione eliminata",
			"session_id", oldest.SessionID,
			"reason", oldest.SaveReason.String(),
			"size_bytes", oldest.SizeBytes,
		)
	}
	return nil
}
```

- [ ] **Step 4.4: Esegui i test — devono passare**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -v -run "TestFileSessionStore_Save|TestFileSessionStore_Load"
```

Expected: tutti `PASS`.

- [ ] **Step 4.5: Commit**

```
git add internal/session/file_store.go internal/session/session_test.go
git commit -m "feat(session): FileSessionStore con Save, Load, LoadByMessageHash"
```

---

## Task 5: FileSessionStore — List, Delete, Evict

**Files:**
- Modifica: `internal/session/session_test.go`

- [ ] **Step 5.1: Aggiungi i test per List, Delete ed Evict**

```go
func TestFileSessionStore_List_SortedByUpdatedAt(t *testing.T) {
	store, _ := newTempStore(t)
	ctx := context.Background()

	stateA := &session.SessionState{
		SessionID:  "A",
		Messages:   []session.Message{{Role: "user", Content: "sessione A"}},
		MusicState: session.MusicState{Key: "Am"},
	}
	stateB := &session.SessionState{
		SessionID:  "B",
		Messages:   []session.Message{{Role: "user", Content: "sessione B"}},
		MusicState: session.MusicState{Key: "Cm"},
	}

	_ = store.Save(ctx, stateA, session.SaveReasonAuto)
	// stateB salvato dopo stateA → deve apparire prima nella lista
	_ = store.Save(ctx, stateB, session.SaveReasonAuto)

	metas, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("attese 2 sessioni, trovate %d", len(metas))
	}
	// ordinato per UpdatedAt discendente: B prima di A
	if metas[0].SessionID != session.MessageHash(stateB.Messages) {
		t.Errorf("prima sessione attesa hash di B, got %s", metas[0].SessionID)
	}
}

func TestFileSessionStore_Delete(t *testing.T) {
	store, dir := newTempStore(t)
	ctx := context.Background()
	state := sampleState("del-session")

	_ = store.Save(ctx, state, session.SaveReasonManual)
	hash := session.MessageHash(state.Messages)

	if err := store.Delete(ctx, hash); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	path := filepath.Join(dir, hash+".czs")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file non eliminato dopo Delete")
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
				{Role: "user", Content: fmt.Sprintf("sessione %d", i)},
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
```

- [ ] **Step 5.2: Esegui i test — devono passare (il codice esiste già)**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -v -run "TestFileSessionStore_List|TestFileSessionStore_Delete|TestFileSessionStore_Evict"
```

Expected: tutti `PASS`.

- [ ] **Step 5.3: Verifica build completa**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go build ./...
```

Expected: zero errori.

- [ ] **Step 5.4: Commit**

```
git add internal/session/session_test.go
git commit -m "test(session): copertura List, Delete, Evict"
```

---

## Task 6: Integrazione config

**Files:**
- Modifica: `internal/config/config.go`
- Test: `internal/config/config_test.go` (aggiunta)

- [ ] **Step 6.1: Aggiungi test per SessionSection in `config_test.go`**

Apri `internal/config/config_test.go` e aggiungi:

```go
func TestLoad_SessionDefaults(t *testing.T) {
	cfg, err := config.Load()
	// Load può fallire se non c'è cadenza.yaml — usiamo i defaults hard-coded
	if err != nil {
		cfg = &config.AppConfig{}
	}
	// Verifica che la sezione Session esista e abbia i defaults
	if cfg.Session.MaxSizeMB == 0 {
		t.Error("Session.MaxSizeMB deve avere un default > 0")
	}
	if cfg.Session.MaxSessions == 0 {
		t.Error("Session.MaxSessions deve avere un default > 0")
	}
	if cfg.Session.CheckpointInterval == 0 {
		t.Error("Session.CheckpointInterval deve avere un default > 0")
	}
}
```

- [ ] **Step 6.2: Esegui il test — deve fallire**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/config/... -v -run TestLoad_SessionDefaults
```

Expected: `cfg.Session.MaxSizeMB` è zero (campo non esiste).

- [ ] **Step 6.3: Modifica `internal/config/config.go`**

Aggiungi `SessionSection` dopo `CacheSection`:

```go
// SessionSection parametrizza la persistenza delle sessioni su disco.
type SessionSection struct {
	Dir                string `mapstructure:"dir"`
	MaxSizeMB          int    `mapstructure:"max_size_mb"`
	CheckpointInterval int    `mapstructure:"checkpoint_interval"`
	MaxSessions        int    `mapstructure:"max_sessions"`
}
```

Aggiungi il campo `Session` ad `AppConfig` (dopo `Cache CacheSection`):

```go
Session SessionSection `mapstructure:"session"`
```

Aggiungi i default in `setDefaults`, dopo il blocco Cache:

```go
// Session
home, _ := os.UserHomeDir()
v.SetDefault("session.dir", filepath.Join(home, ".cadenza", "sessions"))
v.SetDefault("session.max_size_mb", 512)
v.SetDefault("session.checkpoint_interval", 3)
v.SetDefault("session.max_sessions", 20)
```

Aggiungi `"os"` e `"path/filepath"` agli import (sono già presenti — verifica).

- [ ] **Step 6.4: Esegui il test — deve passare**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/config/... -v -run TestLoad_SessionDefaults
```

Expected: `PASS`.

- [ ] **Step 6.5: Build completa**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go build ./...
```

- [ ] **Step 6.6: Commit**

```
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): aggiunge SessionSection con defaults persistenza sessioni"
```

---

## Task 7: CLI — flag, resume, list-sessions, shutdown hook

**Files:**
- Modifica: `cmd/cadenza/main.go`

- [ ] **Step 7.1: Aggiungi i flag di sessione in `main()` dentro `cmd/cadenza/main.go`**

Dopo `nonInteractiveFlag` (riga ~86), aggiungi:

```go
resumeFlag        := flag.Bool("resume", false, "Riprendi l'ultima sessione salvata")
resumeIDFlag      := flag.String("resume-id", "", "ID sessione specifica da riprendere (SHA1 hash)")
listSessionsFlag  := flag.Bool("list-sessions", false, "Elenca le sessioni salvate e termina")
saveSessionFlag   := flag.Bool("save", false, "Salva la sessione corrente su disco e termina")
sessionDirFlag    := flag.String("session-dir", "", "Directory per le sessioni (default: ~/.cadenza/sessions)")
checkpointFlag    := flag.Int("checkpoint-interval", 0, "Salva automaticamente ogni N generazioni (0 = usa config)")
```

- [ ] **Step 7.2: Aggiungi la gestione `--list-sessions` in `main()`, prima del blocco `flagMode`**

Dopo `setupLogger(...)` (intorno a riga 107), aggiungi:

```go
// Costruisce SessionConfig (merge config + flag)
sessionCfg := buildSessionConfig(appCfg, *sessionDirFlag, *checkpointFlag)
store := session.NewFileStore(sessionCfg)

if *listSessionsFlag {
    printSessionList(store)
    return
}

if *resumeFlag || *resumeIDFlag != "" {
    resumeSession(store, *resumeIDFlag)
    return
}
```

- [ ] **Step 7.3: Aggiungi lo shutdown hook in `runGeneration()`**

In `runGeneration()`, dopo `ctx, cancel := signal.NotifyContext(...)`, aggiungi il salvataggio shutdown:

```go
// Salva la sessione allo shutdown (Ctrl+C)
go func() {
    <-ctx.Done()
    if store != nil {
        shutdownState := buildShutdownState(cfg, seedStr, prog)
        _ = store.Save(context.Background(), shutdownState, session.SaveReasonShutdown)
    }
}()
```

Ma `store` non è disponibile in `runGeneration`. Il modo più pulito è passarlo come parametro oppure renderlo globale.

**Approccio**: crea una variabile package-level `activeStore *session.FileSessionStore` (aggiunta a `var lastRun lastRunInfo`):

```go
var activeStore *session.FileSessionStore
```

Poi in `main()` dopo `store := session.NewFileStore(sessionCfg)`:

```go
activeStore = store
```

- [ ] **Step 7.4: Aggiungi le funzioni helper nel file `cmd/cadenza/main.go`**

Aggiungi alla fine del file:

```go
// buildSessionConfig costruisce SessionConfig da config + flag CLI.
func buildSessionConfig(appCfg *config.AppConfig, dirOverride string, intervalOverride int) session.SessionConfig {
    cfg := session.SessionConfig{
        Dir:                appCfg.Session.Dir,
        MaxSizeMB:          appCfg.Session.MaxSizeMB,
        CheckpointInterval: appCfg.Session.CheckpointInterval,
        MaxSessions:        appCfg.Session.MaxSessions,
    }
    if dirOverride != "" {
        cfg.Dir = dirOverride
    }
    if intervalOverride > 0 {
        cfg.CheckpointInterval = intervalOverride
    }
    return cfg
}

// printSessionList stampa la lista delle sessioni ordinata per UpdatedAt.
func printSessionList(store *session.FileSessionStore) {
    metas, err := store.List(context.Background())
    if err != nil {
        fmt.Fprintf(os.Stderr, "  %s✗  Errore lettura sessioni: %v%s\n", ansiRed, err, ansiReset)
        return
    }
    if len(metas) == 0 {
        fmt.Printf("  %sNessuna sessione salvata.%s\n", ansiDim, ansiReset)
        return
    }

    fmt.Printf("\n  %sSESSIONI SALVATE%s\n\n", ansiDim+ansiWhite, ansiReset)
    for i, m := range metas {
        fmt.Printf("  %s[%d]%s  %s%-44s%s  %s%s%s  msg:%-3d  pat:%-3d  %s%.1f KB%s\n",
            ansiYellow+ansiBold, i+1, ansiReset,
            ansiBold, m.SessionID[:min(16, len(m.SessionID))]+"...", ansiReset,
            ansiCyan, m.UpdatedAt.Format("2006-01-02 15:04"), ansiReset,
            m.MessageCount, m.PatternCount,
            ansiDim, float64(m.SizeBytes)/1024, ansiReset,
        )
    }
    fmt.Printf("\n  %sRiprendi con:%s  cadenza --resume-id <id>\n\n", ansiDim, ansiReset)
}

// resumeSession carica e stampa il riepilogo di una sessione salvata.
func resumeSession(store *session.FileSessionStore, sessionID string) {
    ctx := context.Background()

    var (
        state *session.SessionState
        err   error
    )

    if sessionID != "" {
        state, err = store.Load(ctx, sessionID)
    } else {
        metas, listErr := store.List(ctx)
        if listErr != nil || len(metas) == 0 {
            fmt.Printf("  %sNessuna sessione da riprendere.%s\n", ansiDim, ansiReset)
            return
        }
        state, err = store.Load(ctx, metas[0].SessionID)
    }

    if err != nil {
        fmt.Fprintf(os.Stderr, "  %s✗  Resume fallito: %v%s\n", ansiRed, err, ansiReset)
        return
    }

    ms := state.MusicState
    fmt.Printf("\n  %sRiprendo sessione %s%s%s\n", ansiGreen+ansiBold, ansiReset, state.SessionID[:min(16, len(state.SessionID))], "...")
    fmt.Printf("  %sKey:%s   %s   %sBPM:%s %d\n", ansiBold, ansiReset, ms.Key, ansiBold, ansiReset, ms.BPM)
    fmt.Printf("  %sPattern:%s  %d   %sStep evolutivo:%s %d\n", ansiBold, ansiReset, len(ms.Patterns), ansiBold, ansiReset, ms.EvolutionStep)
    fmt.Printf("  %sUltimo salvataggio:%s  %s  (%s)\n\n",
        ansiBold, ansiReset,
        state.UpdatedAt.Format("2006-01-02 15:04"),
        state.SaveReason.String(),
    )

    // Continua la generazione con i parametri della sessione ripristinata
    cfg := cliConfig{
        BPM:          float64(ms.BPM),
        Key:          ms.Key,
        OutputDir:    defaultOutputDir(),
        NoLLM:        true,
        Bars:         16,
        Variations:   1,
        Groove:       "straight",
        OfflineStyle: "",
        Interactive:  false,
    }
    if err := ensureWritableDir(cfg.OutputDir); err != nil {
        fmt.Fprintf(os.Stderr, "  %s✗  %v%s\n", ansiRed, err, ansiReset)
        return
    }
    runGeneration(cfg)
}
```

- [ ] **Step 7.5: Aggiungi l'import `session` in `cmd/cadenza/main.go`**

Aggiungi all'import block:

```go
"github.com/Andrea-Cavallo/cadenza/internal/session"
```

- [ ] **Step 7.6: Aggiungi il checkpoint dopo ogni generazione riuscita**

In `runSingleGeneration`, prima di `return nil` finale (dopo riga ~648), aggiungi:

```go
// Checkpoint automatico
if activeStore != nil {
    checkpointState := buildCheckpointState(cfg, seedStr, prog, outputFiles)
    if err := activeStore.Save(context.Background(), checkpointState, session.SaveReasonAuto); err != nil {
        slog.Warn("checkpoint sessione fallito", "err", err)
    }
}
```

Aggiungi la funzione helper:

```go
// buildCheckpointState costruisce un SessionState con lo stato corrente della generazione.
func buildCheckpointState(cfg cliConfig, seed string, prog theory.ChordProgression, files []string) *session.SessionState {
    chords := make([]session.StoredChord, len(prog.Chords))
    for i, c := range prog.Chords {
        chords[i] = session.StoredChord{Root: c.Root, Quality: string(c.Quality), Bars: c.Bars}
    }
    patterns := make([]session.Pattern, len(files))
    for i, f := range files {
        patterns[i] = session.Pattern{ID: filepath.Base(f), StepIndex: i, CreatedAt: time.Now()}
    }
    return &session.SessionState{
        SessionID: seed,
        Messages: []session.Message{
            {Role: "system", Content: fmt.Sprintf("key=%s bpm=%.0f seed=%s", cfg.Key, cfg.BPM, seed)},
        },
        MusicState: session.MusicState{
            Key:           cfg.Key,
            BPM:           int(cfg.BPM),
            TimeSignature: "4/4",
            Patterns:      patterns,
            HarmonyHistory: []session.StoredProgression{{
                Key: prog.Key, Mode: prog.Mode, Chords: chords,
            }},
            EvolutionStep: len(files),
        },
    }
}

// buildShutdownState è come buildCheckpointState ma senza files (chiamato da signal handler).
func buildShutdownState(cfg cliConfig, seed string, prog theory.ChordProgression) *session.SessionState {
    return buildCheckpointState(cfg, seed, prog, nil)
}
```

- [ ] **Step 7.7: Build e verifica zero errori**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go build ./...
go vet ./...
```

Expected: zero errori.

- [ ] **Step 7.8: Esegui test esistenti — nessuna regressione**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./... -count=1
```

Expected: tutti `PASS`.

- [ ] **Step 7.9: Smoke test CLI**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go run ./cmd/cadenza/ --list-sessions
```

Expected: `Nessuna sessione salvata.` (prima generazione).

```
go run ./cmd/cadenza/ --bpm 122 --key Am --no-llm
go run ./cmd/cadenza/ --list-sessions
```

Expected: lista con 1 sessione (auto-checkpoint dopo generazione).

```
go run ./cmd/cadenza/ --resume
```

Expected: stampa riepilogo sessione + genera nuovi file.

- [ ] **Step 7.10: Commit**

```
git add cmd/cadenza/main.go
git commit -m "feat(cli): flag --resume, --list-sessions, --save, checkpoint automatico, shutdown hook"
```

---

## Task 8: Coverage e qualità

- [ ] **Step 8.1: Coverage sul package session**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
go test ./internal/session/... -coverprofile=coverage_session.out
go tool cover -func=coverage_session.out | grep -E "total|session"
```

Expected: coverage ≥ 80%.

- [ ] **Step 8.2: Lint**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
golangci-lint run ./internal/session/... ./internal/config/... ./cmd/cadenza/...
```

Expected: zero `Error:`.

- [ ] **Step 8.3: Cross-compile Linux**

```
cd C:\Users\Andrea\Desktop\midillmnew-master\midillm_go-master\backend
GOOS=linux go build ./...
```

Expected: zero errori.

- [ ] **Step 8.4: Commit finale**

```
git add .
git commit -m "test(session): coverage ≥80%, lint pulito, cross-compile Linux OK"
```

---

## Self-review — copertura spec

| Requisito spec | Task che lo implementa |
|----------------|------------------------|
| Header binario 32 byte (magic CZS, version, save_reason, counts, timestamps) | Task 2 |
| Naming file = SHA1 messaggi + `.czs` | Task 3, Task 4 |
| `SessionStore` interface con 6 metodi | Task 1 |
| `FileSessionStore` implementa l'interfaccia | Task 4, 5 |
| `SessionMeta` con tutti i campi | Task 1, Task 4 |
| Eviction FIFO per count e size | Task 5 |
| `SessionConfig` con defaults | Task 1, Task 6 |
| Flag `--resume`, `--resume-id`, `--list-sessions` | Task 7 |
| Checkpoint automatico dopo ogni generazione | Task 7 |
| Shutdown hook su SIGINT | Task 7 |
| `SessionSection` in `AppConfig` | Task 6 |
| Log in italiano | Task 4 (`slog.Info("sessione salvata"...)`) |
| Nessuna compressione, nessuna encryption | KISS — non implementato per scelta |
| Test coverage ≥ 80% | Task 8 |

---

**Piano completo e salvato.**
