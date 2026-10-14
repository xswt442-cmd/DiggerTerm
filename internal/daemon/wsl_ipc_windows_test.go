//go:build windows

package daemon

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWSLObservedOverDaemonIPC(t *testing.T) {
	if os.Getenv("DIGGERTERM_WSL_OBSERVE_TESTS") != "1" {
		t.Skip("set DIGGERTERM_WSL_OBSERVE_TESTS=1 for real WSL IPC integration")
	}
	distro := os.Getenv("DIGGERTERM_WSL_TEST_DISTRO")
	if distro == "" {
		distro = "Ubuntu"
	}
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "wsl.exe")
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("WSL unavailable: %v", err)
	}

	t.Setenv("DIGGERTERM_IPC_NAMESPACE", fmt.Sprintf("wsl-ipc-%d", os.Getpid()))
	listener, err := listenLocal()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	go func() { _ = server.Serve() }()
	defer server.shutdown()

	linuxShell := exec.Command(exe, "-d", distro, "--exec", "/bin/sh", "-c", "echo $$; sleep 22 & wait")
	stdout, err := linuxShell.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := linuxShell.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = linuxShell.Process.Kill(); _ = linuxShell.Wait() }()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	targetPID, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("invalid Linux PID %q: %v", line, err)
	}

	client, err := Dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	deadline := time.Now().Add(17 * time.Second)
	for time.Now().Before(deadline) {
		snap, err := client.Observe()
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range snap.Processes {
			if item.Source == "wsl" && item.Distribution == distro &&
				item.PID == targetPID && item.Kind == "terminal" && strings.HasPrefix(item.Key, "wsl:"+distro+":") {
				t.Logf("WSL process visible through daemon IPC: %+v", item)
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("WSL %s running shell was not returned by observe IPC", distro)
}
