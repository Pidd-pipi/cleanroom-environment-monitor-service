package domain

import "fmt"

// IsoLimit is the ISO 14644-1 maximum permitted particle concentration
// (particles per cubic metre) for a given particle size class.
type IsoLimit struct {
	// Count0303 is the limit for particles >= 0.3 micrometre.
	Count0303 float64
	// Count0505 is the limit for particles >= 0.5 micrometre.
	Count0505 float64
}

// Rank returns the order of an ISO class where smaller numbers are cleaner.
// It is used to compare "worse" classes when both particle sizes are judged.
func (c IsoClass) Rank() int {
	switch c {
	case Iso5:
		return 5
	case Iso6:
		return 6
	case Iso7:
		return 7
	case Iso8:
		return 8
	default:
		return 99
	}
}

// WorseThan reports whether c is a dirtier class than other.
func (c IsoClass) WorseThan(other IsoClass) bool {
	return c.Rank() > other.Rank()
}

// ClassifyISO determines the ISO class from particle concentrations using
// the dual-size rule: each size is classified independently and the worse
// (dirtier) class wins. A concentration above every supplied limit maps to
// a class worse than ISO 8 (returned as the "below_iso8" sentinel via the
// boolean). limitTable must contain entries for the returned classes.
func ClassifyISO(limitTable map[IsoClass]IsoLimit, count0303, count0505 float64) (IsoClass, bool) {
	// Start from the cleanest supported class and relax until both sizes fit.
	best := Iso5
	overTable := false
	for _, cls := range AllIsoClasses() {
		lim, ok := limitTable[cls]
		if !ok {
			continue
		}
		if count0303 <= lim.Count0303 && count0505 <= lim.Count0505 {
			best = cls
			break
		}
		best = cls
	}
	// If even the dirtiest class in the table is exceeded, report the
	// sentinel over-table flag.
	last := limitTable[Iso8]
	if count0303 > last.Count0303 || count0505 > last.Count0505 {
		overTable = true
	}
	return best, overTable
}

// RatioAgainst computes the worst (maximum) concentration-to-limit ratio of
// a sample against a monitor zone threshold. A ratio >= 1 means the zone
// exceeded its configured particle limit.
func RatioAgainst(lim IsoLimit, count0303, count0505 float64) float64 {
	r1, r2 := 0.0, 0.0
	if lim.Count0303 > 0 {
		r1 = count0303 / lim.Count0303
	}
	if lim.Count0505 > 0 {
		r2 = count0505 / lim.Count0505
	}
	if r1 >= r2 {
		return r1
	}
	return r2
}

// String returns a human readable label of the ISO class.
func (c IsoClass) String() string {
	return string(c)
}

// ValidateLimitTable ensures the supplied table covers all supported classes
// with positive limits.
func ValidateLimitTable(table map[IsoClass]IsoLimit) error {
	for _, cls := range AllIsoClasses() {
		lim, ok := table[cls]
		if !ok {
			return fmt.Errorf("domain: missing ISO limit for %s", cls)
		}
		if lim.Count0303 <= 0 || lim.Count0505 <= 0 {
			return fmt.Errorf("domain: non-positive ISO limit for %s", cls)
		}
	}
	return nil
}
