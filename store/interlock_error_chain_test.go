package store

import (
	"errors"
	"testing"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// TestInterlockUpdateMissingErrorsIsNotFound verifies updating a missing
// interlock log surfaces a not_found domain error.
func TestInterlockUpdateMissingErrorsIsNotFound(t *testing.T) {
	st := NewMemoryStore()
	log := domain.NewInterlockLog("nope", "z1", "PA-A", "m1", []string{"z1"}, domain.InterlockFFUSpeedUp, 1, "manual", 2.0)
	_, err := st.Interlocks().Update(log)
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("update of missing log must unwrap to a domain error, got %v", err)
	}
	if de.Code != domain.CodeNotFound {
		t.Fatalf("want not_found, got %s", de.Code)
	}
}

// TestInterlockGetMissingErrorsIsNotFound verifies fetching a missing
// interlock log surfaces a not_found domain error.
func TestInterlockGetMissingErrorsIsNotFound(t *testing.T) {
	st := NewMemoryStore()
	_, err := st.Interlocks().Get("nope")
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("get of missing log must unwrap to a domain error, got %v", err)
	}
	if de.Code != domain.CodeNotFound {
		t.Fatalf("want not_found, got %s", de.Code)
	}
}
