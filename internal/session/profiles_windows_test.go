package session

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

// Opt-in integration probes: run with DIGGERTERM_EXTENDED_PTY_TESTS=1.
// They require PowerShell and an initialized default WSL distribution.
func TestExtendedWindowsProfiles(t *testing.T) {
	if os.Getenv("DIGGERTERM_EXTENDED_PTY_TESTS") != "1" {
		t.Skip("set DIGGERTERM_EXTENDED_PTY_TESTS=1 for PowerShell and WSL probes")
	}
	cases := []struct {
		profile string
		command string
		want    string
	}{
		{profile: "powershell", command: "Write-Output ('DIGGERTERM_PS_' + (2+3))\r\n", want: "DIGGERTERM_PS_5"},
		{profile: "wsl", command: "printf 'DIGGERTERM_WSL_%s\\n' \"$(uname -s)\"\r\n", want: "DIGGERTERM_WSL_Linux"},
	}
	for _, tc := range cases {
		t.Run(tc.profile, func(t *testing.T) {
			m := NewManager(nil, nil)
			info, err := m.Create(tc.profile, 90, 30)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			defer func() { _ = m.Terminate(info.ID) }()
			if err := m.Write(info.ID, tc.command); err != nil {
				t.Fatalf("Write: %v", err)
			}
			deadline := time.Now().Add(30 * time.Second)
			for time.Now().Before(deadline) {
				reply, err := m.Snapshot(info.ID)
				if err != nil {
					t.Fatalf("Snapshot: %v", err)
				}
				data, err := base64.StdEncoding.DecodeString(reply.Data)
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				if strings.Contains(string(data), tc.want) {
					return
				}
				time.Sleep(80 * time.Millisecond)
			}
			t.Fatalf("timed out waiting for %q from %s", tc.want, tc.profile)
		})
	}
}
