package permission

import (
	"context"
	"sync"
	"testing"
)

func TestAskOnceMemo_FirstCallAsks(t *testing.T) {
	p := NewAskOnceMemo()
	d, _ := p.Decide(context.Background(), "read_file", `{"path":"main.go"}`)
	if d != DecisionAsk {
		t.Fatalf("first call: want DecisionAsk, got %v", d)
	}
}

func TestAskOnceMemo_AfterRememberAllows(t *testing.T) {
	p := NewAskOnceMemo()
	p.Remember("read_file")
	d, _ := p.Decide(context.Background(), "read_file", `{"path":"main.go"}`)
	if d != DecisionAllow {
		t.Fatalf("after Remember: want DecisionAllow, got %v", d)
	}
	// other tool still asks
	d2, _ := p.Decide(context.Background(), "bash", `{"command":"ls"}`)
	if d2 != DecisionAsk {
		t.Fatalf("unrelated tool: want DecisionAsk, got %v", d2)
	}
}

func TestAskOnceMemo_ConcurrentSafe(t *testing.T) {
	p := NewAskOnceMemo()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); p.Remember("grep") }()
		go func() {
			defer wg.Done()
			p.Decide(context.Background(), "grep", `{"pattern":"x"}`)
		}()
	}
	wg.Wait()
	if d, _ := p.Decide(context.Background(), "grep", ""); d != DecisionAllow {
		t.Fatalf("after concurrent Remembers: want DecisionAllow, got %v", d)
	}
}
