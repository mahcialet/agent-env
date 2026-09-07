package domain

import (
	"testing"
	"time"
)

func TestImmutableSourceTuple(t *testing.T) {
	a := Source{Alias: "api", RepositoryID: "one", Commit: "aaa", RequestedRef: "main"}
	b := Source{Alias: "ui", RepositoryID: "two", Commit: "bbb"}
	want := SourceDigest([]Source{a, b})
	a.RequestedRef = "feature"
	a.WorktreePath = "a different checkout"
	if got := SourceDigest([]Source{b, a}); got != want {
		t.Fatalf("mutable metadata or order changed digest: %s != %s", got, want)
	}
	a.Commit = "ccc"
	if SourceDigest([]Source{a, b}) == want {
		t.Fatal("changed pinned commit must change digest")
	}
}

func TestStateMachineAndQuarantine(t *testing.T) {
	l := Lease{Observed: "requested"}
	for _, next := range []string{"allocating", "starting", "ready", "degraded", "releasing", "quarantined", "releasing", "released"} {
		if err := Transition(&l, next); err != nil {
			t.Fatal(err)
		}
	}
	if err := Transition(&l, "ready"); err == nil {
		t.Fatal("released resources cannot become ready without allocation")
	}
	if l.Observed != "released" {
		t.Fatal("failed transition mutated state")
	}
}

func TestGCSafety(t *testing.T) {
	now := time.Now()
	l := Lease{Desired: "active", Observed: "ready", ExpiresAt: now.Add(-10 * time.Minute), HeartbeatAt: now.Add(-2 * time.Minute)}
	if !GCEligible(l, now) {
		t.Fatal("expired ready lease should be candidate")
	}
	for _, state := range []string{"allocating", "starting", "releasing", "quarantined"} {
		l.Observed = state
		if GCEligible(l, now) {
			t.Fatalf("in-flight/unsafe %s automatically collected", state)
		}
	}
	l.Observed = "ready"
	l.ExpiresAt = now.Add(time.Hour)
	if GCEligible(l, now) {
		t.Fatal("live lease collected")
	}
}

func TestGCGraceBoundaries(t *testing.T) {
	now := time.Now()
	l := Lease{Desired: "active", Observed: "ready", ExpiresAt: now.Add(-5 * time.Minute), HeartbeatAt: now.Add(-time.Minute)}
	if !GCEligible(l, now) {
		t.Fatal("elapsed grace boundaries should qualify")
	}
	l.ExpiresAt = now.Add(-5*time.Minute + time.Nanosecond)
	if GCEligible(l, now) {
		t.Fatal("expiry grace not elapsed")
	}
	l.ExpiresAt = now.Add(-10 * time.Minute)
	l.HeartbeatAt = now.Add(-time.Minute + time.Nanosecond)
	if GCEligible(l, now) {
		t.Fatal("recent heartbeat ignored")
	}
	l.ExpiresAt = time.Time{}
	if GCEligible(l, now) {
		t.Fatal("missing expiry cannot prove collection eligibility")
	}
}
