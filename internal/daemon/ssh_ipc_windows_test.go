//go:build windows

package daemon

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

func TestSSHIPCIsDaemonOwned(t *testing.T) {
	t.Setenv("DIGGERTERM_IPC_NAMESPACE", fmt.Sprintf("ssh-%d", os.Getpid()))
	listener, err := listenLocal()
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	defer server.shutdown()

	// A TCP endpoint that immediately closes the connection. This exercises
	// SSH process creation via IPC without contacting any real remote server
	// or changing ~/.ssh/known_hosts.
	remote, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	go func() {
		c, err := remote.Accept()
		if err == nil {
			_ = c.Close()
		}
	}()

	first, err := Dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.CreateSSH("-oStrictHostKeyChecking=no", 0, 80, 24); err == nil {
		t.Fatal("IPC accepted SSH option injection")
	}
	if _, err := first.CreateSSH("host", 65536, 80, 24); err == nil {
		t.Fatal("IPC accepted invalid SSH port")
	}
	if _, err := first.Create("ssh", 80, 24); err == nil {
		t.Fatal("SSH must not be created through the local profile method")
	}
	info, err := first.CreateSSH("127.0.0.1", remote.Addr().(*net.TCPAddr).Port, 80, 24)
	if err != nil {
		t.Fatalf("IPC SSH start: %v", err)
	}
	defer func() {
		for _, item := range server.manager.List() {
			if item.Status == "running" {
				_ = server.manager.Terminate(item.ID)
			}
		}
	}()
	if info.ID == "" || info.PID < 1 || info.Profile != "ssh" || info.Name != "SSH · 127.0.0.1" {
		t.Fatalf("wrong SSH session metadata: %+v", info)
	}
	_ = first.Close()
	second, err := Dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	items, err := second.List()
	if err != nil || len(items) != 1 || items[0].ID != info.ID || items[0].PID != info.PID {
		t.Fatalf("SSH was not preserved after GUI/client disconnect: %+v err=%v", items, err)
	}

	deadline := time.Now().Add(14 * time.Second)
	for time.Now().Before(deadline) {
		items, err = second.List()
		if err == nil && len(items) == 1 && items[0].Status == "exited" {
			break
		}
		time.Sleep(45 * time.Millisecond)
	}
	if len(items) != 1 || items[0].Status != "exited" {
		t.Fatalf("SSH did not finish after server rejected connection: %+v", items)
	}
	if err := second.Forget(info.ID); err != nil {
		t.Fatal(err)
	}
	if err := second.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("daemon failed to stop after SSH ended")
	}
}
