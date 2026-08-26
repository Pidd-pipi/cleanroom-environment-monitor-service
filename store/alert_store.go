package store

import (
	"sort"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// AlertStore is the repository for CleanAlert entities.
type AlertStore struct{ s *Store }

// Alerts returns the alert repository.
func (s *Store) Alerts() *AlertStore { return &AlertStore{s: s} }

// Create inserts an alert.
func (r *AlertStore) Create(a domain.CleanAlert) (domain.CleanAlert, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.state.Alerts = append(r.s.state.Alerts, a)
	return a, r.s.flushLocked()
}

// Update replaces an alert (used by ack/escalate/close).
func (r *AlertStore) Update(a domain.CleanAlert) (domain.CleanAlert, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for i := range r.s.state.Alerts {
		if r.s.state.Alerts[i].ID == a.ID {
			r.s.state.Alerts[i] = a
			return a, r.s.flushLocked()
		}
	}
	return domain.CleanAlert{}, domain.NotFound("alert", a.ID)
}

// Get returns an alert by id.
func (r *AlertStore) Get(id string) (domain.CleanAlert, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	for _, a := range r.s.state.Alerts {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.CleanAlert{}, domain.NotFound("alert", id)
}

// List returns alerts optionally filtered by status/type, newest first.
func (r *AlertStore) List(filter AlertFilter) ([]domain.CleanAlert, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.CleanAlert, 0)
	for _, a := range r.s.state.Alerts {
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Type != "" && a.Type != filter.Type {
			continue
		}
		if filter.MonitorZoneID != "" && a.MonitorZoneID != filter.MonitorZoneID {
			continue
		}
		if filter.CleanZoneID != "" && a.CleanZoneID != filter.CleanZoneID {
			continue
		}
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

// AlertFilter filters alert listings.
type AlertFilter struct {
	Status        domain.AlertStatus
	Type          domain.AlertType
	MonitorZoneID string
	CleanZoneID   string
}

// FindOpenByDedupKey returns the most recent open/acknowledged alert with
// the given dedup key (used by the dedup-window merge).
func (r *AlertStore) FindOpenByDedupKey(dedupKey string) (domain.CleanAlert, bool, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	var best domain.CleanAlert
	found := false
	for _, a := range r.s.state.Alerts {
		if a.DedupKey != dedupKey {
			continue
		}
		if a.Status == domain.AlertStatusClosed {
			continue
		}
		if !found || a.CreatedAt.After(best.CreatedAt) {
			best = a
			found = true
		}
	}
	return best, found, nil
}

// CountActive returns the number of alerts still needing attention.
func (r *AlertStore) CountActive() (int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	n := 0
	for _, a := range r.s.state.Alerts {
		if a.IsActive() {
			n++
		}
	}
	return n, nil
}

// CountByCleanZoneActive returns active alerts grouped per clean zone.
func (r *AlertStore) CountByCleanZoneActive() (map[string]int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := map[string]int{}
	for _, a := range r.s.state.Alerts {
		if a.IsActive() {
			out[a.CleanZoneID]++
		}
	}
	return out, nil
}

// ListOverdue returns open alerts older than the escalation window.
func (r *AlertStore) ListOverdue(now time.Time, window time.Duration) ([]domain.CleanAlert, error) {
	all, err := r.List(AlertFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]domain.CleanAlert, 0)
	for _, a := range all {
		if a.NeedsEscalation(now, window) {
			out = append(out, a)
		}
	}
	return out, nil
}
