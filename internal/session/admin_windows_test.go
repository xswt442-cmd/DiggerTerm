//go:build windows

package session

import (
	"os"
	"testing"

	"github.com/xswt442-cmd/DiggerTerm/internal/winsecurity"
	"golang.org/x/sys/windows"
)

func TestAdministratorSessionInheritsElevation(t *testing.T) {
	elevated, err := winsecurity.Elevated()
	if err != nil {
		t.Fatal(err)
	}
	if !elevated {
		if os.Getenv("DIGGERTERM_REQUIRE_ELEVATED_TEST") == "1" {
			t.Fatal("test requires an elevated process")
		}
		t.Skip("requires an elevated process")
	}
	manager := NewManager(nil, nil)
	info, err := manager.Create("cmd", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Terminate(info.ID)
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(info.PID))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	if !token.IsElevated() {
		t.Fatal("managed cmd did not inherit elevated token")
	}
}
