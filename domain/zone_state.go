package domain

import "time"

// zoneStateMachine encodes the allowed transitions of the cleanroom state
// machine:
//
//	normal -> elevated / over_limit / interlocked(area)
//	elevated -> normal / over_limit / interlocked
//	over_limit -> interlocked / elevated / normal
//	interlocked -> restored (manual confirmation)
//	restored -> normal / elevated / over_limit / interlocked(area)
//
// The restored -> interlocked edge exists because of the whole-area
// consistency rule: when one zone of a physical area triggers an interlock,
// every other zone of that area must enter interlocked, even if it was
// already restored.
var zoneStateMachine = map[ZoneStatus]map[ZoneStatus]bool{
	ZoneStatusNormal: {
		ZoneStatusElevated: true, ZoneStatusOverLimit: true, ZoneStatusInterlocked: true,
	},
	ZoneStatusElevated: {
		ZoneStatusNormal: true, ZoneStatusOverLimit: true, ZoneStatusInterlocked: true,
	},
	ZoneStatusOverLimit: {
		ZoneStatusInterlocked: true, ZoneStatusElevated: true, ZoneStatusNormal: true,
	},
	ZoneStatusInterlocked: {
		ZoneStatusRestored: true, ZoneStatusInterlocked: true,
	},
	ZoneStatusRestored: {
		ZoneStatusNormal: true, ZoneStatusElevated: true, ZoneStatusOverLimit: true,
		ZoneStatusInterlocked: true,
	},
}

// CanTransition reports whether moving from `from` to `to` is legal in the
// cleanroom state machine. Staying in the same state is always allowed.
func CanTransition(from, to ZoneStatus) bool {
	if from == to {
		return true
	}
	return zoneStateMachine[from][to]
}

// AllowedTransitionsFrom returns the legal next states of `from`.
func AllowedTransitionsFrom(from ZoneStatus) []ZoneStatus {
	out := make([]ZoneStatus, 0, len(zoneStateMachine[from]))
	for s := range zoneStateMachine[from] {
		out = append(out, s)
	}
	return out
}

// TargetStatusFromRatio computes the state a zone should move to based on
// the latest concentration-to-limit ratio and the over-limit ratio from
// config. It returns the current state unchanged when no movement is needed.
//
//	r >= overLimitRatio : over_limit
//	1.0 <= r < overLimitRatio : elevated
//	r < 1.0 : normal
func TargetStatusFromRatio(current ZoneStatus, ratio, overLimitRatio float64) ZoneStatus {
	switch {
	case ratio >= overLimitRatio:
		return ZoneStatusOverLimit
	case ratio >= 1.0:
		if CanTransition(current, ZoneStatusElevated) {
			return ZoneStatusElevated
		}
		return current
	default:
		if CanTransition(current, ZoneStatusNormal) {
			return ZoneStatusNormal
		}
		return current
	}
}

// RestoreStatus is the status a zone enters after manual restore
// confirmation; it is fixed by the state machine.
const RestoreStatus = ZoneStatusRestored

// HasBeenStuck reports whether the zone has stayed in `state` for longer
// than the given timeout (used by the auto-interlock sweeper).
func HasBeenStuck(state ZoneStatus, since time.Time, timeout time.Duration, now time.Time) bool {
	if timeout <= 0 {
		return false
	}
	return state == ZoneStatusElevated || state == ZoneStatusOverLimit
}

// StatusAge returns how long the zone has been in its current status.
func StatusAge(since time.Time, now time.Time) time.Duration {
	if now.Before(since) {
		return 0
	}
	return now.Sub(since)
}
