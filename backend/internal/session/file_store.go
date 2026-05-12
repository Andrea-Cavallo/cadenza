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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listUnlocked(ctx)
}

func (s *FileSessionStore) listUnlocked(ctx context.Context) ([]SessionMeta, error) {
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
	defer func() { _ = f.Close() }()

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

// MaxSizeMB restituisce il limite massimo in MB configurato per lo store.
func (s *FileSessionStore) MaxSizeMB() int {
	return s.cfg.MaxSizeMB
}

// CheckpointInterval restituisce l'intervallo di checkpoint configurato.
func (s *FileSessionStore) CheckpointInterval() int {
	return s.cfg.CheckpointInterval
}

func (s *FileSessionStore) Evict(ctx context.Context, maxSizeMB int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metas, err := s.listUnlocked(ctx)
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
