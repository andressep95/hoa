package health

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestMonitor_StateTransition(t *testing.T) {
	// Mock alternates: 1st ok, 2nd err, 3rd ok, ...
	var calls atomic.Int32
	original := pingFn
	defer func() { pingFn = original }()
	pingFn = func(_ context.Context, _ string) (bool, error) {
		n := calls.Add(1)
		if n == 1 {
			return true, nil
		}
		return false, errors.New("simulated outage")
	}

	m := NewMonitor()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, "stub-dsn", 30*time.Millisecond)

	// First update: false (initial) -> true. Drain it.
	select {
	case <-m.Updates():
	case <-time.After(2 * time.Second):
		t.Fatalf("did not get initial ok update; calls=%d", calls.Load())
	}
	if ok, _, _ := m.Status(); !ok {
		t.Fatalf("expected ok=true after initial check")
	}

	// Second update: true -> false (mock returns err for n>=2)
	select {
	case <-m.Updates():
	case <-time.After(2 * time.Second):
		t.Fatalf("did not get flip-to-err update; calls=%d", calls.Load())
	}
	ok, err, _ := m.Status()
	if ok {
		t.Fatalf("expected ok=false after flip, got ok")
	}
	if err == nil {
		t.Fatalf("expected non-nil err after flip")
	}
}

func TestMonitor_StableNoExtraEmit(t *testing.T) {
	original := pingFn
	defer func() { pingFn = original }()
	pingFn = func(_ context.Context, _ string) (bool, error) {
		return true, nil
	}

	m := NewMonitor()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx, "stub-dsn", 30*time.Millisecond)

	// Drain first update (false -> true)
	select {
	case <-m.Updates():
	case <-time.After(2 * time.Second):
		t.Fatalf("did not get initial update")
	}

	// After that, with stable ok=true, no further emits expected.
	select {
	case <-m.Updates():
		t.Fatalf("received unexpected update without state transition")
	case <-time.After(200 * time.Millisecond):
	}
}
