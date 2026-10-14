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
	return `\\.\pipe\diggerterm-` + sid + suffix, sid, nil
}

func listenLocal() (net.Listener, error) {
	name, sid, err := pipeNameAndSID()
	if err != nil {
		return nil, err
	}
	// Default named-pipe ACLs can allow undesired remote/local principals.
	// Constrain the channel to this OS user plus SYSTEM, with no inherited DACL.
	cfg := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;" + sid + ")",
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}
	return winio.ListenPipe(name, cfg)
}

func dialLocal(timeout time.Duration) (net.Conn, error) {
	name, _, err := pipeNameAndSID()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return winio.DialPipeContext(ctx, name)
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
