//go:build windows

package observe

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// This starts an ordinary process outside the DiggerTerm Session Manager.
// Discovery must never require owning its ConPTY, reading its arguments,
// or modifying that process.
func TestWindowsExternalProcessReadOnly(t *testing.T) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 15")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("launch external PowerShell: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := Scan(nil)
		if err != nil {
			t.Fatalf("scan Windows processes: %v", err)
		}
		if !snapshot.Supported {
			t.Fatal("Windows scanner is not supported")
		}
		if snapshot.ScannedAt == "" {
			t.Fatal("missing scan time")
		}
		var found bool
		for _, row := range snapshot.Processes {
			if row.PID != cmd.Process.Pid {
				continue
			}
			found = true
			if row.Kind != "terminal" || row.Name != "powershell.exe" || row.ParentPID != os.Getpid() {
				t.Fatalf("unexpected observed metadata: %+v", row)
			}
			if _, err := time.Parse(time.RFC3339Nano, row.StartedAt); err != nil {
				t.Fatalf("invalid process timestamp: %+v: %v", row, err)
			}
		}
		if found {
			excluded, err := Scan([]int{cmd.Process.Pid})
			if err != nil {
				t.Fatalf("scan managed exclusions: %v", err)
			}
			for _, row := range excluded.Processes {
				if row.PID == cmd.Process.Pid {
					t.Fatal("owned process appeared as external")
				}
			}
			// Scanning itself performs no process control.
			return
		}
		time.Sleep(120 * time.Millisecond)
	}
	t.Fatalf("external PowerShell PID %d was not observed", cmd.Process.Pid)
}
