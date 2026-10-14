//go:build windows

package daemon

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIPCReconnectPreservesPTY(t *testing.T) {
	t.Setenv("DIGGERTERM_IPC_NAMESPACE", fmt.Sprintf("test-%d", os.Getpid()))
	listener, err := listenLocal()
	if err != nil {
		t.Fatalf("listen private named pipe: %v", err)
	}
	srv := NewServer(listener)
	done := make(chan error, 1)
	go func() { done <- srv.Serve() }()
	defer srv.shutdown()

	first, err := Dial(nil)
	if err != nil {
		t.Fatalf("dial IPC: %v", err)
	}
	info, err := first.Create("cmd", 80, 24)
	if err != nil {
		t.Fatalf("create daemon-owned terminal: %v", err)
	}
	defer func() {
		// Explicit cleanup even if a test assertion fails.
		if c, dialErr := Dial(nil); dialErr == nil {
			_ = c.Terminate(info.ID)
			_ = c.Close()
		}
	}()
	if info.PID <= 0 {
		t.Fatalf("no shell PID: %+v", info)
	}
	if err := first.Write(info.ID, "echo DIGGERTERM_DAEMON_RECONNECT\r\n"); err != nil {
		t.Fatalf("write to PTY: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("detach client: %v", err)
	}

	// The original client is gone; a fresh GUI/client must see the SAME shell.
	second, err := Dial(nil)
	if err != nil {
		t.Fatalf("reconnect to daemon: %v", err)
	}
	defer second.Close()
	items, err := second.List()
	if err != nil || len(items) != 1 || items[0].PID != info.PID || items[0].Status != "running" {
		t.Fatalf("recovered wrong session: %+v, err=%v", items, err)
	}
	if err := second.Stop(); err == nil {
		t.Fatal("daemon must refuse stop with active PTY")
	}
	// Observe is a separate read-only IPC method. The owned CMD must not
	// appear in the list of external processes.
	observed, err := second.Observe()
	if err != nil {
		t.Fatalf("observe via IPC: %v", err)
	}
	if !observed.Supported {
		t.Fatal("Windows observe should be supported")
	}
	for _, item := range observed.Processes {
		if item.PID == info.PID {
			t.Fatalf("managed shell leaked into external list: %+v", item)
		}
	}
	deadline := time.Now().Add(12 * time.Second)
	for {
		snapshot, err := second.Snapshot(info.ID)
		if err != nil {
			t.Fatalf("snapshot: %v", err)
		}
		output, err := base64.StdEncoding.DecodeString(snapshot.Data)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(output), "DIGGERTERM_DAEMON_RECONNECT") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no PTY output after reconnect")
		}
		time.Sleep(40 * time.Millisecond)
	}
	if err := second.Resize(info.ID, 110, 28); err != nil {
		t.Fatalf("resize after reconnect: %v", err)
	}
	if err := second.Terminate(info.ID); err != nil {
		t.Fatalf("terminate: %v", err)
	}
	for {
		items, err := second.List()
		if err == nil && len(items) == 1 && items[0].Status == "exited" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("shell did not exit: %+v, err=%v", items, err)
		}
		time.Sleep(40 * time.Millisecond)
	}
	if err := second.Forget(info.ID); err != nil {
		t.Fatalf("forget: %v", err)
	}
	if err := second.Stop(); err != nil {
		t.Fatalf("explicit daemon stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("daemon returned error: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("daemon did not stop after explicit action")
	}
}

func TestInvalidIPCNamespace(t *testing.T) {
	t.Setenv("DIGGERTERM_IPC_NAMESPACE", "../other")
	if _, err := endpointSuffix(); err == nil {
		t.Fatal("unsafe namespace should fail")
	}
}
