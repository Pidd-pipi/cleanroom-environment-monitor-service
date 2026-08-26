package domain

import "testing"

// TestClassifyISOCleanStaysClean verifies a clean reading is classified at
// the cleanest ISO class it fits.
func TestClassifyISOCleanStaysClean(t *testing.T) {
	table := map[IsoClass]IsoLimit{
		Iso5: {Count0303: 100000, Count0505: 35000},
		Iso6: {Count0303: 1000000, Count0505: 350000},
		Iso7: {Count0303: 10000000, Count0505: 3500000},
		Iso8: {Count0303: 100000000, Count0505: 35000000},
	}
	cls, over := ClassifyISO(table, 30000, 8000)
	if cls != Iso5 {
		t.Fatalf("clean reading must classify as Iso5, got %s", cls)
	}
	if over {
		t.Fatal("clean reading must not be over_table")
	}
}
