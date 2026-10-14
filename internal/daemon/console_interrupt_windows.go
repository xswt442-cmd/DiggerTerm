//go:build windows

package daemon

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	consoleHandlerAPI = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")
	// Never let a Ctrl+C intended for a daemon-owned ConPTY child terminate
	// the long-running daemon itself. Registering a real handler (rather than
	// SetConsoleCtrlHandler(NULL, TRUE)) does not propagate Ctrl+C ignore state
	// to subsequently spawned console processes.
	daemonCtrlHandler = syscall.NewCallback(func(event uint32) uintptr {
		if event == 0 { // CTRL_C_EVENT
			return 1 // handled by the daemon, not a daemon shutdown
		}
		return 0
	})
)

// The detached daemon is intentionally started WITHOUT CREATE_NEW_PROCESS_GROUP:
// that flag disables native Ctrl+C delivery to foreground ConPTY children.
// Reset any inherited Ctrl+C-ignore state and register a handler that protects
// the daemon itself without passing "ignore Ctrl+C" on to new PTY children.
//
// See Windows console process-group behaviour in Microsoft Learn.
func prepareDaemonConsoleInterrupts() error {
	if ok, _, err := consoleHandlerAPI.Call(0, 0); ok == 0 {
		return fmt.Errorf("re-enable Ctrl+C handling: %w", err)
	}
	if ok, _, err := consoleHandlerAPI.Call(daemonCtrlHandler, 1); ok == 0 {
		return fmt.Errorf("protect daemon console handler: %w", err)
	}
	return nil
}
