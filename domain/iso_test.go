package domain

import "testing"

func TestClassifyISO(t *testing.T) {
	table := map[IsoClass]IsoLimit{
		Iso5: {Count0303: 100000, Count0505: 35000},
		Iso6: {Count0303: 1000000, Count0505: 350000},
		Iso7: {Count0303: 10000000, Count0505: 3500000},
		Iso8: {Count0303: 100000000, Count0505: 35000000},
	}
	cases := []struct {
		name         string
		c0303, c0505 float64
		wantClass    IsoClass
		wantOver     bool
	}{
		{"clean iso5", 30000, 8000, Iso5, false},
		{"boundary iso5", 100000, 35000, Iso5, false},
		{"iso6 by 0.3um", 500000, 40000, Iso6, false},
		{"worse of two sizes", 500000, 400000, Iso7, false},
		{"iso7", 5000000, 1000000, Iso7, false},
		{"iso8", 50000000, 10000000, Iso8, false},
		{"over table 0.3um", 200000000, 10000000, Iso8, true},
		{"over table 0.5um", 10000000, 50000000, Iso8, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cls, over := ClassifyISO(table, tc.c0303, tc.c0505)
			if cls != tc.wantClass {
				t.Fatalf("expected class %s, got %s", tc.wantClass, cls)
			}
			if over != tc.wantOver {
				t.Fatalf("expected overTable=%v, got %v", tc.wantOver, over)
			}
		})
	}
}

func TestRatioAgainst(t *testing.T) {
	lim := IsoLimit{Count0303: 100000, Count0505: 35000}
	cases := []struct {
		name         string
		c0303, c0505 float64
		want         float64
	}{
		{"at limit", 100000, 35000, 1.0},
		{"above via 0.3um", 150000, 35000, 1.5},
		{"above via 0.5um", 100000, 70000, 2.0},
		{"below", 50000, 10000, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RatioAgainst(lim, tc.c0303, tc.c0505); got != tc.want {
				t.Fatalf("expected %.2f, got %.2f", tc.want, got)
			}
		})
	}
}

func TestIsoClassRanking(t *testing.T) {
	if !Iso8.WorseThan(Iso5) {
		t.Fatal("iso8 must be worse than iso5")
	}
	if Iso5.WorseThan(Iso6) {
		t.Fatal("iso5 must not be worse than iso6")
	}
}

func TestValidateLimitTable(t *testing.T) {
	if err := ValidateLimitTable(map[IsoClass]IsoLimit{}); err == nil {
		t.Fatal("empty table must be rejected")
	}
	table := map[IsoClass]IsoLimit{
		Iso5: {Count0303: 1, Count0505: 1},
		Iso6: {Count0303: 1, Count0505: 1},
		Iso7: {Count0303: 1, Count0505: 1},
		Iso8: {Count0303: 1, Count0505: 1},
	}
	if err := ValidateLimitTable(table); err != nil {
		t.Fatalf("valid table rejected: %v", err)
	}
}
