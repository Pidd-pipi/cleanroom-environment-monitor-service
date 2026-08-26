package store

import (
	"fmt"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// CleanZoneStore is the repository for CleanZone entities.
type CleanZoneStore struct{ s *Store }

// CleanZones returns the clean-zone repository.
func (s *Store) CleanZones() *CleanZoneStore { return &CleanZoneStore{s: s} }

// Create inserts a new clean zone. Duplicate ids are rejected.
func (r *CleanZoneStore) Create(z domain.CleanZone) (domain.CleanZone, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if err := z.Validate(); err != nil {
		return domain.CleanZone{}, err
	}
	for i := range r.s.state.CleanZones {
		if r.s.state.CleanZones[i].ID == z.ID {
			return domain.CleanZone{}, domain.Conflict(fmt.Sprintf("clean zone %q already exists", z.ID))
		}
	}
	r.s.state.CleanZones = append(r.s.state.CleanZones, z)
	return z, r.s.flushLocked()
}

// Update replaces an existing clean zone.
func (r *CleanZoneStore) Update(z domain.CleanZone) (domain.CleanZone, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if err := z.Validate(); err != nil {
		return domain.CleanZone{}, err
	}
	for i := range r.s.state.CleanZones {
		if r.s.state.CleanZones[i].ID == z.ID {
			r.s.state.CleanZones[i] = z
			return z, r.s.flushLocked()
		}
	}
	return domain.CleanZone{}, domain.NotFound("clean zone", z.ID)
}

// Get returns a clean zone by id.
func (r *CleanZoneStore) Get(id string) (domain.CleanZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	for i := range r.s.state.CleanZones {
		if r.s.state.CleanZones[i].ID == id {
			return r.s.state.CleanZones[i], nil
		}
	}
	return domain.CleanZone{}, domain.NotFound("clean zone", id)
}

// GetOrErr is an alias used by services for readability.
func (r *CleanZoneStore) GetOrErr(id string) (domain.CleanZone, error) { return r.Get(id) }

// List returns all clean zones (copy).
func (r *CleanZoneStore) List() ([]domain.CleanZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.CleanZone, len(r.s.state.CleanZones))
	copy(out, r.s.state.CleanZones)
	return out, nil
}

// ListByPhysicalArea returns the clean zones sharing a physical area.
func (r *CleanZoneStore) ListByPhysicalArea(area string) ([]domain.CleanZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.CleanZone, 0)
	for _, z := range r.s.state.CleanZones {
		if z.PhysicalArea == area {
			out = append(out, z)
		}
	}
	return out, nil
}

// Delete removes a clean zone; monitor zones belonging to the zone are
// removed too (cascade) so no orphaned monitor points survive the parent.
func (r *CleanZoneStore) Delete(id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	idx := -1
	for i := range r.s.state.CleanZones {
		if r.s.state.CleanZones[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return domain.NotFound("clean zone", id)
	}
	// Drop the clean zone (shift left and reslice, do not leave a stale tail).
	copy(r.s.state.CleanZones[idx:], r.s.state.CleanZones[idx+1:])
	r.s.state.CleanZones = r.s.state.CleanZones[:len(r.s.state.CleanZones)-1]

	// Cascade: drop monitor zones whose parent is the deleted clean zone.
	kept := make([]domain.MonitorZone, 0, len(r.s.state.MonitorZones))
	for _, m := range r.s.state.MonitorZones {
		if m.CleanZoneID == id {
			continue
		}
		kept = append(kept, m)
	}
	r.s.state.MonitorZones = kept
	return r.s.flushLocked()
}

// flushLocked must be called with the write lock held.
func (s *Store) flushLocked() error {
	s.state.Version = stateVersion
	return s.saveLocked()
}

// Count returns the number of clean zones.
func (r *CleanZoneStore) Count() (int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	return len(r.s.state.CleanZones), nil
}
