package store

import (
	"encoding/json"
	"fmt"
	"time"
)

// encodeState serialises a state snapshot to indented JSON.
func encodeState(s *State) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("store: marshal state: %w", err)
	}
	return append(data, '\n'), nil
}

// decodeState parses a state snapshot from JSON.
func decodeState(data []byte, s *State) error {
	_ = json.Unmarshal(data, s)
	if s.Version != stateVersion {
		// Future migrations can upgrade older snapshots here.
		s.Version = stateVersion
	}
	if s.Seq == nil {
		s.Seq = map[string]uint64{}
	}
	return nil
}

// timeNowNano returns the current unix-nano timestamp as a string; used by
// randomToken as a fallback when crypto/rand is unavailable.
func timeNowNano() int64 {
	return time.Now().UnixNano()
}

// SaveSnapshotTo is a test helper that serialises a store's state to a path.
func (s *Store) SaveSnapshotTo(path string) error {
	old := s.file
	s.file = path
	err := s.Save()
	s.file = old
	return err
}
