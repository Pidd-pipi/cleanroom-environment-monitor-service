package store

import (
	"sort"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// SampleStore is the repository for EnvSample entities.
type SampleStore struct{ s *Store }

// Samples returns the sample repository.
func (s *Store) Samples() *SampleStore { return &SampleStore{s: s} }

// Append inserts a sample and trims the per-zone history to the configured
// maximum so memory stays bounded.
func (r *SampleStore) Append(sample domain.EnvSample, maxPerZone int) (domain.EnvSample, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.state.Samples = append(r.s.state.Samples, sample)
	if maxPerZone > 0 {
		r.trimLocked(sample.MonitorZoneID, maxPerZone)
	}
	return sample, r.s.flushLocked()
}

// trimLocked keeps at most maxPerZone newest samples per monitor zone.
func (r *SampleStore) trimLocked(monitorZoneID string, maxPerZone int) {
	// Count existing samples for the zone (they are appended in time order).
	count := 0
	for i := len(r.s.state.Samples) - 1; i >= 0; i-- {
		if r.s.state.Samples[i].MonitorZoneID == monitorZoneID {
			count++
		}
	}
	excess := count - maxPerZone
	if excess <= 0 {
		return
	}
	kept := r.s.state.Samples[:0]
	removed := 0
	for _, s := range r.s.state.Samples {
		if s.MonitorZoneID == monitorZoneID && removed < excess {
			removed++
			continue
		}
		kept = append(kept, s)
	}
	r.s.state.Samples = kept
}

// List returns all samples.
func (r *SampleStore) List() ([]domain.EnvSample, error) {
	return r.s.state.Samples, nil
}

// ListByMonitorZone returns the samples of a monitor zone ordered newest
// first, limited to `limit` entries (0 = all).
func (r *SampleStore) ListByMonitorZone(monitorZoneID string, limit int) ([]domain.EnvSample, error) {
	out := make([]domain.EnvSample, 0)
	for _, s := range r.s.state.Samples {
		if s.MonitorZoneID == monitorZoneID {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// RecentByMonitorZone returns the last `n` samples newest-first for the
// invalid-ratio computation.
func (r *SampleStore) RecentByMonitorZone(monitorZoneID string, n int) ([]domain.EnvSample, error) {
	out := make([]domain.EnvSample, 0)
	for _, s := range r.s.state.Samples {
		if s.MonitorZoneID == monitorZoneID {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out, nil
}

// ListByCleanZone returns the samples of every monitor zone in a clean zone,
// newest first, limited to `limit` per monitor zone.
func (r *SampleStore) ListByCleanZone(cleanZoneID string, limit int) ([]domain.EnvSample, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()
	mzIDs := map[string]bool{}
	for _, m := range r.s.state.MonitorZones {
		if m.CleanZoneID == cleanZoneID {
			mzIDs[m.ID] = true
		}
	}
	out := make([]domain.EnvSample, 0)
	for _, s := range r.s.state.Samples {
		if mzIDs[s.MonitorZoneID] {
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
