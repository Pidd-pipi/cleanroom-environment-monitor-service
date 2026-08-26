package store

import (
	"fmt"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// MonitorZoneStore is the repository for MonitorZone entities.
type MonitorZoneStore struct{ s *Store }

// MonitorZones returns the monitor-zone repository.
func (s *Store) MonitorZones() *MonitorZoneStore { return &MonitorZoneStore{s: s} }

// Create inserts a monitor zone. The owning clean zone must exist and the
// monitor-zone id must be unique.
func (r *MonitorZoneStore) Create(m domain.MonitorZone) (domain.MonitorZone, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if err := m.Validate(); err != nil {
		return domain.MonitorZone{}, err
	}
	zoneExists := false
	for _, z := range r.s.state.CleanZones {
		if z.ID == m.CleanZoneID {
			zoneExists = true
			break
		}
	}
	if !zoneExists {
		return domain.MonitorZone{}, domain.NotFound("clean zone", m.CleanZoneID)
	}
	for _, existing := range r.s.state.MonitorZones {
		if existing.ID == m.ID {
			return domain.MonitorZone{}, domain.Conflict(fmt.Sprintf("monitor zone %q already exists", m.ID))
		}
		if existing.ParticleCounterID == m.ParticleCounterID {
			return domain.MonitorZone{}, domain.Conflict(fmt.Sprintf("particle counter %q already assigned to %q", m.ParticleCounterID, existing.ID))
		}
	}
	r.s.state.MonitorZones = append(r.s.state.MonitorZones, m)
	return m, r.s.flushLocked()
}

// Update replaces a monitor zone.
func (r *MonitorZoneStore) Update(m domain.MonitorZone) (domain.MonitorZone, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	if err := m.Validate(); err != nil {
		return domain.MonitorZone{}, err
	}
	for i := range r.s.state.MonitorZones {
		if r.s.state.MonitorZones[i].ID == m.ID {
			r.s.state.MonitorZones[i] = m
			return m, r.s.flushLocked()
		}
	}
	return domain.MonitorZone{}, domain.NotFound("monitor zone", m.ID)
}

// Get returns a monitor zone by id.
func (r *MonitorZoneStore) Get(id string) (domain.MonitorZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	for _, m := range r.s.state.MonitorZones {
		if m.ID == id {
			return m, nil
		}
	}
	return domain.MonitorZone{}, domain.NotFound("monitor zone", id)
}

// GetOrErr is an alias used by services for readability.
func (r *MonitorZoneStore) GetOrErr(id string) (domain.MonitorZone, error) { return r.Get(id) }

// List returns all monitor zones (copy).
func (r *MonitorZoneStore) List() ([]domain.MonitorZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.MonitorZone, len(r.s.state.MonitorZones))
	copy(out, r.s.state.MonitorZones)
	return out, nil
}

// ListByCleanZone returns the monitor zones of a clean zone.
func (r *MonitorZoneStore) ListByCleanZone(cleanZoneID string) ([]domain.MonitorZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	out := make([]domain.MonitorZone, 0)
	for _, m := range r.s.state.MonitorZones {
		if m.CleanZoneID == cleanZoneID {
			out = append(out, m)
		}
	}
	return out, nil
}

// ListByPhysicalArea returns every monitor zone whose clean zone shares the
// given physical area (used by the whole-area interlock rule).
func (r *MonitorZoneStore) ListByPhysicalArea(area string) ([]domain.MonitorZone, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	zoneIDs := map[string]bool{}
	for _, z := range r.s.state.CleanZones {
		if z.PhysicalArea == area {
			zoneIDs[z.ID] = true
		}
	}
	out := make([]domain.MonitorZone, 0)
	for _, m := range r.s.state.MonitorZones {
		if zoneIDs[m.CleanZoneID] {
			out = append(out, m)
		}
	}
	return out, nil
}

// Count returns the number of monitor zones.
func (r *MonitorZoneStore) Count() (int, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	return len(r.s.state.MonitorZones), nil
}
