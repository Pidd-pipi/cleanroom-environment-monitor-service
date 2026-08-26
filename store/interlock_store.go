package store

import (
	"fmt"
	"sort"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// InterlockStore is the repository for InterlockLog entities.
type InterlockStore struct{ s *Store }

// Interlocks returns the interlock repository.
func (s *Store) Interlocks() *InterlockStore { return &InterlockStore{s: s} }

// Create inserts an interlock log entry.
func (r *InterlockStore) Create(log domain.InterlockLog) (domain.InterlockLog, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.state.Interlocks = append(r.s.state.Interlocks, log)
	return log, r.s.flushLocked()
}

// Update replaces an interlock log entry (used to record restore).
func (r *InterlockStore) Update(log domain.InterlockLog) (domain.InterlockLog, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i := range r.s.state.Interlocks {
		if r.s.state.Interlocks[i].ID == log.ID {
			r.s.state.Interlocks[i] = log
			return log, r.s.flushLocked()
		}
	}
	return domain.InterlockLog{}, fmt.Errorf("interlock store: %v", domain.NotFound("interlock log", log.ID))
}

// Get returns an interlock log by id.
func (r *InterlockStore) Get(id string) (domain.InterlockLog, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	for _, l := range r.s.state.Interlocks {
		if l.ID == id {
			return l, nil
		}
	}
	return domain.InterlockLog{}, fmt.Errorf("interlock store: %v", domain.NotFound("interlock log", id))
}

// List returns all interlock logs newest first.
func (r *InterlockStore) List() ([]domain.InterlockLog, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.InterlockLog, len(r.s.state.Interlocks))
	copy(out, r.s.state.Interlocks)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out, nil
}

// ListByCleanZone returns interlock logs for a clean zone newest first.
func (r *InterlockStore) ListByCleanZone(cleanZoneID string) ([]domain.InterlockLog, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.InterlockLog, 0)
	for _, l := range r.s.state.Interlocks {
		if l.CleanZoneID == cleanZoneID {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out, nil
}

// OpenForCleanZone returns the open (unrestored) interlock logs of a zone.
func (r *InterlockStore) OpenForCleanZone(cleanZoneID string) ([]domain.InterlockLog, error) {
	all, err := r.ListByCleanZone(cleanZoneID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.InterlockLog, 0)
	for _, l := range all {
		if l.IsOpen() {
			out = append(out, l)
		}
	}
	return out, nil
}

// OpenCount returns how many interlock events are still active overall.
func (r *InterlockStore) OpenCount() (int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	n := 0
	for _, l := range r.s.state.Interlocks {
		if l.IsOpen() {
			n++
		}
	}
	return n, nil
}
