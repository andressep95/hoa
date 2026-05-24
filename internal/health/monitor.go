// Package health implements lightweight liveness probes for external
// dependencies (currently Oracle). The Monitor runs in a background goroutine
// and exposes its latest verdict via Status(); subscribers consume Updates()
// to react to state changes (typically the TUI footer).
package health

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/sijms/go-ora/v2" // register oracle driver
)

// Monitor pings a DSN periodically and tracks its liveness.
type Monitor struct {
	mu        sync.Mutex
	ok        bool
	lastErr   error
	lastCheck time.Time
	updates   chan struct{}
}

// NewMonitor returns a Monitor in the unknown state. Start it with Start().
func NewMonitor() *Monitor {
	return &Monitor{updates: make(chan struct{}, 8)}
}

// Status returns the latest verdict.
func (m *Monitor) Status() (ok bool, err error, since time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ok, m.lastErr, m.lastCheck
}

// Updates returns a channel that emits when the OK state flips.
func (m *Monitor) Updates() <-chan struct{} { return m.updates }

// Start runs the monitoring loop in a goroutine. It performs an immediate
// check, then ticks every interval. The loop exits when ctx is cancelled.
func (m *Monitor) Start(ctx context.Context, dsn string, interval time.Duration) {
	go m.run(ctx, dsn, interval)
}

func (m *Monitor) run(ctx context.Context, dsn string, interval time.Duration) {
	m.tick(ctx, dsn) // immediate first check

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.tick(ctx, dsn)
		}
	}
}

func (m *Monitor) tick(ctx context.Context, dsn string) {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	prevOK, _, _ := m.Status()
	ok, err := pingFn(pingCtx, dsn)

	m.mu.Lock()
	m.ok = ok
	m.lastErr = err
	m.lastCheck = time.Now()
	m.mu.Unlock()

	if prevOK != ok {
		select {
		case m.updates <- struct{}{}:
		default:
		}
	}
}

// pingFn opens a short-lived connection to the DSN and pings it. Indirected
// through a package var so tests can substitute a mock.
var pingFn = func(ctx context.Context, dsn string) (bool, error) {
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return false, err
	}
	return true, nil
}
