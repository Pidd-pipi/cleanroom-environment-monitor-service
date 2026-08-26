package httpapi

import (
	"net/http"
	"testing"
)

// TestAlertListInvalidStatusRejected verifies an unknown alert status
// filter is rejected instead of being silently ignored and returning the
// full unfiltered list.
func TestAlertListInvalidStatusRejected(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, _ := doJSON(t, http.MethodGet, ts.URL+"/api/alerts?status=bogus", nil)
	if res.StatusCode == http.StatusOK {
		t.Fatal("invalid status filter must not be silently ignored")
	}
}
