//go:build !windows

package daemon

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func socketPath() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "DiggerTerm")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("daemon socket directory not private: %s", dir)
	}
	suffix, err := endpointSuffix()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session"+suffix+".sock"), nil
}

type lockedListener struct {
	net.Listener
	lock *os.File
}

func (l *lockedListener) Close() error {
	err := l.Listener.Close()
	_ = syscall.Flock(int(l.lock.Fd()), syscall.LOCK_UN)
	_ = l.lock.Close()
	return err
}

func listenLocal() (net.Listener, error) {
	path, err := socketPath()
	if err != nil {
		return nil, err
	}
	// Serialize daemon startup. Without a lock, a concurrent starter could
	// delete a socket that a different daemon has just opened.
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("another daemon already owns the endpoint: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			_ = lock.Close()
		}
	}()
	if fi, statErr := os.Lstat(path); statErr == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("IPC path is not a socket: %s", path)
		}
		// The process holding this lock must have exited; this is a stale socket.
		c, dialErr := net.DialTimeout("unix", path, 150*time.Millisecond)
		if dialErr == nil {
			_ = c.Close()
			return nil, fmt.Errorf("daemon already responding")
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(statErr) {
		return nil, statErr
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = l.Close()
		return nil, err
	}
	ok = true
	return &lockedListener{Listener: l, lock: lock}, nil
}

func dialLocal(timeout time.Duration) (net.Conn, error) {
	path, err := socketPath()
	if err != nil {
		return nil, err
	}
	return net.DialTimeout("unix", path, timeout)
}

func startDetached() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "--daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
