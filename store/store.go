// Package store provides the persistence layer: in-memory repositories
// backed by an optional JSON file snapshot. The store is safe for
// concurrent use and every mutation can be flushed to disk atomically.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// stateVersion is bumped whenever the persisted schema changes.
const stateVersion = 1

// State is the complete persisted snapshot of the service.
type State struct {
	Version      int                   `json:"version"`
	CleanZones   []domain.CleanZone    `json:"clean_zones"`
	MonitorZones []domain.MonitorZone  `json:"monitor_zones"`
	Samples      []domain.EnvSample    `json:"samples"`
	Interlocks   []domain.InterlockLog `json:"interlocks"`
	Alerts       []domain.CleanAlert   `json:"alerts"`
	AuditEntries []domain.AuditEntry   `json:"audit_entries"`
	Seq          map[string]uint64     `json:"seq"`
}

// Store is the single repository aggregate. Each entity repository lives in
// its own file (cleanzone_store.go, monitor_store.go, ...) but shares the
// same locked state.
type Store struct {
	mu    sync.RWMutex
	state State
	file  string // JSON persistence file; empty disables persistence.

	// loadWarning describes a non-fatal degraded load (e.g. a corrupt file
	// was backed up and the service started with an empty state).
	loadWarning error
}

// NewStore creates a store. When file is non-empty the store loads any
// existing snapshot from it and flushes every mutation to it.
func NewStore(file string) *Store {
	s := &Store{file: file}
	s.state.Version = stateVersion
	s.state.Seq = map[string]uint64{}
	return s
}

// NewMemoryStore creates a store without any file persistence (tests,
// ephemeral runs).
func NewMemoryStore() *Store { return NewStore("") }

// File returns the configured persistence file.
func (s *Store) File() string { return s.file }

// SetFile switches the persistence file. An empty path disables persistence.
func (s *Store) SetFile(file string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.file = file
}

// Load reads the JSON snapshot from the persistence file into memory. It is
// a no-op when the file does not exist. A corrupt file is renamed to
// "<file>.bak" and the service continues with an empty state; the warning is
// exposed through LoadWarning so the caller can log it.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == "" {
		return nil
	}
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("store: read %s: %w", s.file, err)
	}
	if err := decodeState(data, &s.state); err != nil {
		backup := s.file + ".bak"
		if rbErr := os.Rename(s.file, backup); rbErr != nil {
			return fmt.Errorf("store: decode %s: %w (and backup failed: %v)", s.file, err, rbErr)
		}
		s.state = State{Version: stateVersion, Seq: map[string]uint64{}}
		s.loadWarning = fmt.Errorf("store: %s was corrupt (%v); backed up to %s and started with empty state", s.file, err, backup)
		return nil
	}
	if s.state.Seq == nil {
		s.state.Seq = map[string]uint64{}
	}
	return nil
}

// LoadWarning returns the non-fatal warning produced by Load, if any.
func (s *Store) LoadWarning() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadWarning
}

// Save flushes the in-memory state to the persistence file atomically
// (temp file + fsync + rename + directory fsync). Writers are serialised by
// the store write lock so two concurrent Saves cannot race on the temp file.
// It is a no-op when persistence is disabled.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

// saveLocked writes the snapshot; the caller must hold the write lock.
func (s *Store) saveLocked() error {
	data, err := encodeState(&s.state)
	if err != nil {
		return err
	}
	if s.file == "" {
		return nil
	}
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("store: mkdir: %w", err)
	}

	// Write to a unique temp file in the same directory, fsync it, then
	// atomically replace the target. This guarantees readers never observe a
	// partially written snapshot and that the rename is crash-consistent.
	tmp, err := os.CreateTemp(dir, filepath.Base(s.file)+".tmp-*")
	if err != nil {
		return fmt.Errorf("store: create tmp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("store: write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("store: fsync tmp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("store: chmod tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("store: close tmp: %w", err)
	}
	if err := os.Rename(tmpName, s.file); err != nil {
		return fmt.Errorf("store: rename: %w", err)
	}

	// Best-effort directory fsync so the rename itself is durable.
	if dirF, err := os.Open(dir); err == nil {
		_ = dirF.Sync()
		_ = dirF.Close()
	}
	return nil
}

// nextID generates the next monotonic id for a prefix (e.g. "zone_7").
func (s *Store) nextID(prefix string) string {
	s.state.Seq[prefix]++
	return fmt.Sprintf("%s_%d", prefix, s.state.Seq[prefix])
}

// randomToken returns a short random token used for request/event ids.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", timeNowNano())
	}
	return hex.EncodeToString(b)
}

// NewID returns the next monotonic id for a prefix (thread-safe).
func (s *Store) NewID(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextID(prefix)
}

// NewToken returns a random hex token used for request/event correlation.
func (s *Store) NewToken(n int) string { return randomToken(n) }

// snapshot returns a defensive copy of the whole state for read paths that
// need consistent multi-entity views (overview aggregation).
func (s *Store) snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.state
	out.CleanZones = append([]domain.CleanZone(nil), s.state.CleanZones...)
	out.MonitorZones = append([]domain.MonitorZone(nil), s.state.MonitorZones...)
	out.Samples = append([]domain.EnvSample(nil), s.state.Samples...)
	out.Interlocks = append([]domain.InterlockLog(nil), s.state.Interlocks...)
	out.Alerts = append([]domain.CleanAlert(nil), s.state.Alerts...)
	out.AuditEntries = append([]domain.AuditEntry(nil), s.state.AuditEntries...)
	out.Seq = map[string]uint64{}
	for k, v := range s.state.Seq {
		out.Seq[k] = v
	}
	return out
}

// CloneState returns a defensive copy of the state (exported for tests).
func (s *Store) CloneState() State { return s.snapshot() }

// Reset clears all data (used by bootstrap for idempotent seeding).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = State{Version: stateVersion, Seq: map[string]uint64{}}
	s.loadWarning = nil
}
