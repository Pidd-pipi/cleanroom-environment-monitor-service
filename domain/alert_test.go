package domain

import (
	"testing"
	"time"
)

func TestDedupKeyOf(t *testing.T) {
	got := DedupKeyOf("monitor_a", AlertParticle)
	if got != "monitor_a:particle" {
		t.Fatalf("unexpected dedup key %q", got)
	}
}

func TestAlertAckRequiresDisposition(t *testing.T) {
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelCritical, "high", "", time.Now())
	if err := a.Ack("eng", "", time.Now()); err == nil {
		t.Fatal("ack without disposition must fail")
	}
	if err := a.Ack("eng", "cleaned filter", time.Now()); err != nil {
		t.Fatalf("ack with disposition must succeed: %v", err)
	}
	if a.Status != AlertStatusAcknowledged {
		t.Fatalf("expected acknowledged, got %s", a.Status)
	}
}

func TestAlertAckClosed(t *testing.T) {
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelCritical, "high", "", time.Now())
	a.Close(time.Now())
	if err := a.Ack("eng", "note", time.Now()); err == nil {
		t.Fatal("ack of closed alert must fail")
	}
}

func TestAlertEscalateAndClose(t *testing.T) {
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelWarning, "high", "", time.Now())
	a.Escalate(time.Now())
	if a.Status != AlertStatusEscalated {
		t.Fatalf("expected escalated, got %s", a.Status)
	}
	a.Close(time.Now())
	if a.Status != AlertStatusClosed {
		t.Fatalf("expected closed, got %s", a.Status)
	}
}

func TestAlertNeedsEscalation(t *testing.T) {
	now := time.Now()
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelWarning, "high", "", now.Add(-2*time.Hour))
	if !a.NeedsEscalation(now, time.Hour) {
		t.Fatal("open alert older than window must need escalation")
	}
	a.Ack("eng", "done", now)
	if a.NeedsEscalation(now, time.Hour) {
		t.Fatal("acknowledged alert must not need escalation")
	}
	a2 := NewAlert("a2", "z1", "m1", AlertParticle, AlertLevelWarning, "high", "", now.Add(-time.Minute))
	if a2.NeedsEscalation(now, time.Hour) {
		t.Fatal("recent alert must not need escalation")
	}
}

func TestAlertMerge(t *testing.T) {
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelCritical, "first", "s1", time.Now())
	a.Merge(NewAlert("a2", "z1", "m1", AlertParticle, AlertLevelCritical, "second", "s2", time.Now()))
	if a.Count != 2 {
		t.Fatalf("expected count 2, got %d", a.Count)
	}
	if a.Message != "second" {
		t.Fatalf("expected merged message, got %q", a.Message)
	}
}

func TestAlertIsActive(t *testing.T) {
	a := NewAlert("a1", "z1", "m1", AlertParticle, AlertLevelWarning, "m", "", time.Now())
	if !a.IsActive() {
		t.Fatal("open alert must be active")
	}
	a.Escalate(time.Now())
	if !a.IsActive() {
		t.Fatal("escalated alert must be active")
	}
	a.Close(time.Now())
	if a.IsActive() {
		t.Fatal("closed alert must not be active")
	}
}
