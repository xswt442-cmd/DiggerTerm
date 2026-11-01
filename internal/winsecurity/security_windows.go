//go:build windows

// Package winsecurity enforces the UAC boundary for local IPC.
package winsecurity

import (
	"errors"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

func tokenElevated(token windows.Token) (bool, error) {
	var value, size uint32
	err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&value)), uint32(unsafe.Sizeof(value)), &size)
	if err != nil {
		return false, err
	}
	if size != 4 {
		return false, errors.New("invalid elevation token data")
	}
	return value != 0, nil
}

func Elevated() (bool, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false, err
	}
	defer token.Close()
	return tokenElevated(token)
}

func PipeName(base string, elevated bool) string {
	if elevated {
		return base + "-elevated"
	}
	return base
}

func PipeSecurity(sid string, elevated bool) string {
	if elevated {
		return "D:P(A;;GA;;;SY)(A;;GA;;;BA)"
	}
	return "D:P(A;;GA;;;SY)(A;;GA;;;" + sid + ")"
}

// VerifyElevatedPeer rejects a lower-privilege or different-user pipe endpoint,
// including a pipe created by an unprivileged process before the real server.
func VerifyElevatedPeer(conn net.Conn, server bool, expectedSID string) error {
	f, ok := conn.(interface{ Fd() uintptr })
	if !ok {
		return errors.New("named pipe handle unavailable")
	}
	var pid uint32
	var err error
	if server {
		err = windows.GetNamedPipeServerProcessId(windows.Handle(f.Fd()), &pid)
	} else {
		err = windows.GetNamedPipeClientProcessId(windows.Handle(f.Fd()), &pid)
	}
	if err != nil {
		return fmt.Errorf("identify pipe peer: %w", err)
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return fmt.Errorf("open pipe peer: %w", err)
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return fmt.Errorf("open pipe peer token: %w", err)
	}
	defer token.Close()
	elevated, err := tokenElevated(token)
	if err != nil || !elevated {
		return errors.New("pipe peer is not elevated")
	}
	identity, err := token.GetTokenUser()
	if err != nil || identity.User.Sid.String() != expectedSID {
		return errors.New("pipe peer belongs to a different user")
	}
	return nil
}

type elevatedListener struct {
	net.Listener
	sid string
}

func (l *elevatedListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if VerifyElevatedPeer(conn, false, l.sid) == nil {
			return conn, nil
		}
		_ = conn.Close()
	}
}
func ProtectListener(listener net.Listener, sid string, elevated bool) net.Listener {
	if elevated {
		return &elevatedListener{Listener: listener, sid: sid}
	}
	return listener
}
