//go:build windows

package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"time"

	winio "github.com/Microsoft/go-winio"
	"github.com/xswt442-cmd/DiggerTerm/internal/winsecurity"
)

func pipeNameAndSID() (string, string, error) {
	current, err := user.Current()
	if err != nil {
		return "", "", err
	}
	// On Windows, user.Uid is the SID of the current account.
	sid := current.Uid
	if sid == "" {
		return "", "", fmt.Errorf("current user has no SID")
	}
	suffix, err := endpointSuffix()
	if err != nil {
		return "", "", err
	}
	elevated, err := winsecurity.Elevated()
	if err != nil {
		return "", "", fmt.Errorf("inspect process elevation: %w", err)
	}
	return winsecurity.PipeName(`\\.\pipe\diggerterm-`+sid+suffix, elevated), sid, nil
}

func listenLocal() (net.Listener, error) {
	name, sid, err := pipeNameAndSID()
	if err != nil {
		return nil, err
	}
	// Default named-pipe ACLs can allow undesired remote/local principals.
	// Constrain the channel to this OS user plus SYSTEM, with no inherited DACL.
	elevated, err := winsecurity.Elevated()
	if err != nil {
		return nil, err
	}
	cfg := &winio.PipeConfig{
		SecurityDescriptor: winsecurity.PipeSecurity(sid, elevated),
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}
	listener, err := winio.ListenPipe(name, cfg)
	if err != nil {
		return nil, err
	}
	return winsecurity.ProtectListener(listener, sid, elevated), nil
}

func dialLocal(timeout time.Duration) (net.Conn, error) {
	name, sid, err := pipeNameAndSID()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, name)
	if err != nil {
		return nil, err
	}
	elevated, err := winsecurity.Elevated()
	if err == nil && elevated {
		err = winsecurity.VerifyElevatedPeer(conn, true, sid)
	}
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func startDetached() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "--daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags:    0x00000008, // DETACHED_PROCESS; preserve ConPTY console Ctrl+C delivery
		HideWindow:       true,
		NoInheritHandles: true,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
