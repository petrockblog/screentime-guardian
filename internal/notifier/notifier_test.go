package notifier

import (
	"context"
	"testing"
)

func TestNewChain(t *testing.T) {
	chain := NewChain()

	if len(chain.notifiers) != 0 {
		t.Errorf("Expected 0 notifiers in new chain, got %d", len(chain.notifiers))
	}
}

func TestChainAdd(t *testing.T) {
	chain := NewChain()
	log := NewLogNotifier()

	chain.Add(log)

	if len(chain.notifiers) != 1 {
		t.Errorf("Expected 1 notifier after Add, got %d", len(chain.notifiers))
	}
}

func TestLogNotifier(t *testing.T) {
	notifier := NewLogNotifier()
	ctx := context.Background()

	err := notifier.SendWarning(ctx, "testuser", 5)
	if err != nil {
		t.Errorf("Expected no error from LogNotifier, got %v", err)
	}

	err = notifier.SendLockNotice(ctx, "testuser")
	if err != nil {
		t.Errorf("Expected no error from LogNotifier, got %v", err)
	}

	err = notifier.SendTimeExtended(ctx, "testuser", 30)
	if err != nil {
		t.Errorf("Expected no error from LogNotifier, got %v", err)
	}
}

func TestGetUrgency(t *testing.T) {
	tests := []struct {
		minutesLeft int
		expected    string
	}{
		{10, "normal"},
		{5, "normal"},
		{2, "normal"},
		{1, "critical"},
		{0, "critical"},
	}

	for _, tt := range tests {
		result := getUrgency(tt.minutesLeft)
		if result != tt.expected {
			t.Errorf("getUrgency(%d) = %s, expected %s", tt.minutesLeft, result, tt.expected)
		}
	}
}
