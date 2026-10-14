//go:build windows

package observe

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func platformSnapshot() ([]processRow, bool, error) {
	current, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, true, fmt.Errorf("identify current user: %w", err)
	}
	userSID := current.User.Sid.String()

	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, true, fmt.Errorf("enumerate processes: %w", err)
	}
	defer windows.CloseHandle(h)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(h, &entry); err != nil {
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			return nil, true, nil
		}
		return nil, true, err
	}

	var result []processRow
	for {
		if entry.ProcessID != 0 {
			if row, ok := processMetadata(int(entry.ProcessID), int(entry.ParentProcessID), windows.UTF16ToString(entry.ExeFile[:]), userSID); ok {
				result = append(result, row)
			}
		}
		if err := windows.Process32Next(h, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, true, fmt.Errorf("iterate processes: %w", err)
		}
	}
	return result, true, nil
}

func processMetadata(pid, parent int, name, sid string) (processRow, bool) {
	// Query only metadata. No PROCESS_VM_READ, DEBUG privilege or control rights.
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return processRow{}, false // other user, protected or short-lived process
	}
	defer windows.CloseHandle(h)

	var token windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); err != nil {
		return processRow{}, false
	}
	user, err := token.GetTokenUser()
	token.Close()
	if err != nil || user.User.Sid.String() != sid {
		return processRow{}, false
	}

	var created, exited, kernel, userTime windows.Filetime
	if err := windows.GetProcessTimes(h, &created, &exited, &kernel, &userTime); err != nil {
		return processRow{}, false
	}
	instant := time.Unix(0, created.Nanoseconds()).UTC()
	if instant.IsZero() || instant.After(time.Now().Add(10*time.Second)) {
		return processRow{}, false
	}
	return processRow{pid: pid, parentPID: parent, name: name, started: instant}, true
}
