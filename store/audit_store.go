package store

import (
	"sort"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// AuditStore is the repository for AuditEntry records.
type AuditStore struct{ s *Store }

// Audit returns the audit repository.
func (s *Store) Audit() *AuditStore { return &AuditStore{s: s} }

// Create appends an audit entry. Persistence of audit entries is batched
// (they are flushed together with the next mutation) to keep the hot ingest
// path cheap; a manual Flush is provided for tests.
func (r *AuditStore) Create(e domain.AuditEntry) (domain.AuditEntry, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.state.AuditEntries = append(r.s.state.AuditEntries, e)
	return e, r.s.flushLocked()
}

// List returns audit entries newest first, limited to `limit` (0 = all).
func (r *AuditStore) List(limit int, action domain.AuditAction) ([]domain.AuditEntry, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.AuditEntry, 0)
	for _, e := range r.s.state.AuditEntries {
		if action != "" && e.Action != action {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Count returns the total number of audit records.
func (r *AuditStore) Count() (int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	return len(r.s.state.AuditEntries), nil
}

// ListByAction returns audit entries of a specific action (used by tests
// to assert cross-cutting audit coverage).
func (r *AuditStore) ListByAction(action domain.AuditAction) ([]domain.AuditEntry, error) {
	return r.List(0, action)
}
