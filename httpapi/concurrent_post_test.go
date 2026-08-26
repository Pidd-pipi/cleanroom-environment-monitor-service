package httpapi

import (
	"net/http"
	"sync"
	"testing"
)

// TestConcurrentPostSamplesAppendOnce verifies two concurrent sample POSTs
// persist exactly two extra samples (one per request).
func TestConcurrentPostSamplesAppendOnce(t *testing.T) {
	ts, st, _ := newTestServer(t)
	before, err := st.Samples().ListByMonitorZone("monitor_a1", 0)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]interface{}{
		"count_0_3um":   30000,
		"count_0_5um":   8000,
		"temperature":   21.0,
		"humidity":      45.0,
		"pressure_diff": 20.0,
		"timestamp":     "",
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, _ := doJSON(t, http.MethodPost, ts.URL+"/api/monitors/monitor_a1/samples", body)
			if res.StatusCode != http.StatusOK {
				t.Errorf("POST sample: want 200, got %d", res.StatusCode)
			}
		}()
	}
	close(start)
	wg.Wait()

	after, err := st.Samples().ListByMonitorZone("monitor_a1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+2 {
		t.Fatalf("two POSTs must persist exactly two extra samples, got %d -> %d", len(before), len(after))
	}
}
