package daemon

import (
	"encoding/json"
	"fmt"
	"time"
)

// EnsureClient attaches to the current user's daemon; if none responds,
// it starts another instance of this same executable in --daemon mode.
// Multiple simultaneous GUI launches are safe: only one IPC listener wins.
func EnsureClient(handler func(string, json.RawMessage)) (*Client, error) {
	if c, err := Dial(handler); err == nil {
		return c, nil
	}
	if err := startDetached(); err != nil {
		return nil, fmt.Errorf("launch background session daemon: %w", err)
	}
	deadline := time.Now().Add(8 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		c, err := Dial(handler)
		if err == nil {
			return c, nil
		}
		last = err
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("background session daemon did not become ready: %w", last)
}
