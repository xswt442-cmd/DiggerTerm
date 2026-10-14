//go:build windows

package daemon

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateLocalWorkingDirectoryOverIPC(t *testing.T) {
	t.Setenv("DIGGERTERM_IPC_NAMESPACE", fmt.Sprintf("local-dir-%d", os.Getpid()))
	listener, err := listenLocal()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	go func() { _ = server.Serve() }()
	defer server.shutdown()

	client, err := Dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	dir := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateLocal("cmd", "relative", 80, 24); err == nil {
		t.Fatal("accepted relative working directory")
	}
	if _, err := client.CreateLocal("wsl", dir, 80, 24); err == nil {
		t.Fatal("WSL should not receive Windows working directory")
	}
	info, err := client.CreateLocal("cmd", dir, 80, 24)
	if err != nil {
		t.Fatalf("create local cmd in custom directory: %v", err)
	}
	defer func() { _ = client.Terminate(info.ID) }()
	if err := client.Write(info.ID, "echo %CD%\r\n"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		reply, err := client.Snapshot(info.ID)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := base64.StdEncoding.DecodeString(reply.Data)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(decoded)), strings.ToLower(dir)) {
			// Do not need to stop daemon or lose the real terminal session
			// to prove the IPC passed the child working directory correctly.
			return
		}
		time.Sleep(75 * time.Millisecond)
	}
	t.Fatalf("daemon-owned shell did not start in %q", dir)
}
