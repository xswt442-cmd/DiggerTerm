//go:build windows

package observe

import (
	"bufio"
	"encoding/binary"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestParseRunningWSLDistros(t *testing.T) {
	input := "Ubuntu\r\ndocker-desktop\r\nDebian\r\nUbuntu\r\n"
	u16 := utf16.Encode([]rune(input))
	data := make([]byte, len(u16)*2+2)
	data[0], data[1] = 0xff, 0xfe
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(data[2+2*i:], v)
	}
	got := parseRunningDistros(data)
	if strings.Join(got, ",") != "Ubuntu,Debian" {
		t.Fatalf("bad distro decoding: %q", got)
	}
}

func TestLiveWSLReadOnlyProcess(t *testing.T) {
	if os.Getenv("DIGGERTERM_WSL_OBSERVE_TESTS") != "1" {
		t.Skip("set DIGGERTERM_WSL_OBSERVE_TESTS=1 to run WSL2 integration tests")
	}
	distro := os.Getenv("DIGGERTERM_WSL_TEST_DISTRO")
	if distro == "" {
		distro = "Ubuntu"
	}
	exe, err := wslExecutable()
	if err != nil {
		t.Skipf("WSL missing: %v", err)
	}

	// Create an ordinary Linux shell in a real distribution, outside the
	// Session Manager. Its lifetime is self-limited, even if this test exits.
	cmd := exec.Command(exe, "-d", distro, "--exec", "/bin/sh", "-c", "echo $$; sleep 24 & wait")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("get external Linux PID: %v", err)
	}
	targetPID, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("invalid Linux PID %q: %v", line, err)
	}

	deadline := time.Now().Add(16 * time.Second)
	var captured Process
	for time.Now().Before(deadline) {
		rows, _, note := platformWSLSnapshot()
		for _, row := range rows {
			if row.Source == "wsl" && row.Distribution == distro &&
				row.PID == targetPID {
				captured = row
				break
			}
		}
		if captured.Key != "" {
			break
		}
		t.Logf("await WSL process: %s", note)
		time.Sleep(200 * time.Millisecond)
	}
	if captured.Key == "" {
		t.Fatalf("failed to observe Linux process inside %s", distro)
	}
	if !strings.HasPrefix(captured.Key, "wsl:"+distro+":") {
		t.Fatalf("unstable WSL identity: %+v", captured)
	}
	if _, err := time.Parse(time.RFC3339Nano, captured.StartedAt); err != nil {
		t.Fatalf("bad Linux start time: %+v, %v", captured, err)
	}
	t.Logf("read-only WSL process: %+v", captured)
}
