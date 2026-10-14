//go:build windows

package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

func TestRenameBroadcastsToOtherGUIClients(t *testing.T) {
	t.Setenv("DIGGERTERM_IPC_NAMESPACE", fmt.Sprintf("rename-%d", os.Getpid()))
	listener, err := listenLocal()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	go func() { _ = server.Serve() }()
	defer server.shutdown()

	writer, err := Dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	info, err := writer.Create("cmd", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Terminate(info.ID) }()

	received := make(chan session.Info, 2)
	observer, err := Dial(func(kind string, raw json.RawMessage) {
		if kind != "terminal:renamed" {
			return
		}
		var changed session.Info
		if json.Unmarshal(raw, &changed) == nil {
			select {
			case received <- changed:
			default:
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer observer.Close()
	name := "New name / 研发"
	returned, err := writer.Rename(info.ID, name)
	if err != nil || returned.Name != name {
		t.Fatalf("rename failed: %+v %v", returned, err)
	}
	select {
	case got := <-received:
		if got.ID != info.ID || got.Name != name || got.PID != info.PID {
			t.Fatalf("other GUI received wrong metadata: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("other GUI did not receive session rename event")
	}
}
