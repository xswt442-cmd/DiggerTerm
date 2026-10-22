package daemon

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

func TestPeerSessionControlLeaseAndDisconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve() }()
	t.Cleanup(func() {
		server.shutdown()
		select {
		case <-serveDone:
		case <-time.After(3 * time.Second):
			t.Error("test daemon did not stop")
		}
	})

	first := dialTestClient(t, listener.Addr().String())
	second := dialTestClient(t, listener.Addr().String())
	info, err := first.Create(session.Profiles()[0].ID, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = first.Terminate(info.ID)
		waitForClientSessionExit(t, first, info.ID)
	})
	secondInfo, err := first.Create(session.Profiles()[0].ID, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = first.Terminate(secondInfo.ID)
		waitForClientSessionExit(t, first, secondInfo.ID)
	})
	blockedLease := server.sessionLease(info.ID)
	blockedLease.mu.Lock()
	leaseHeld := true
	defer func() {
		if leaseHeld {
			blockedLease.mu.Unlock()
		}
	}()
	writeDone := make(chan error, 1)
	go func() { writeDone <- first.Write(info.ID, "echo queued write\n") }()
	if !waitForPeerPending(t, server, first, writeDone) {
		blockedLease.mu.Unlock()
		leaseHeld = false
		t.Fatal("write request did not enter the session queue")
	}
	resizeDone := make(chan error, 1)
	go func() { resizeDone <- first.Resize(secondInfo.ID, 88, 29) }()
	listDone := make(chan error, 1)
	go func() {
		items, listErr := first.List()
		if listErr == nil && (len(items) < 2 || items[0].ID == "") {
			listErr = errors.New("List returned no managed sessions while another write was blocked")
		}
		listDone <- listErr
	}()
	for label, result := range map[string]<-chan error{"resize on another session": resizeDone, "List on the same GUI connection": listDone} {
		select {
		case err := <-result:
			if err != nil {
				blockedLease.mu.Unlock()
				leaseHeld = false
				t.Fatalf("%s blocked-write regression: %v", label, err)
			}
		case <-time.After(time.Second):
			blockedLease.mu.Unlock()
			leaseHeld = false
			t.Fatalf("%s waited behind another session's blocked write", label)
		}
	}
	select {
	case err := <-writeDone:
		blockedLease.mu.Unlock()
		leaseHeld = false
		t.Fatalf("write completed while its session lease was held: %v", err)
	default:
	}
	blockedLease.mu.Unlock()
	leaseHeld = false
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("queued write failed after lease release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("queued write did not complete after lease release")
	}
	if err := first.Detach(secondInfo.ID); err != nil {
		t.Fatal(err)
	}
	if err := first.Attach(info.ID); err != nil {
		t.Fatalf("first GUI could not attach: %v", err)
	}
	if _, err := second.Snapshot(info.ID); err != nil {
		t.Fatalf("read-only snapshot was denied: %v", err)
	}
	if err := first.Resize(info.ID, 91, 31); err != nil {
		t.Fatalf("owner resize failed: %v", err)
	}
	if err := second.Write(info.ID, "echo denied\n"); err == nil || !strings.Contains(err.Error(), "session controlled by another GUI") {
		t.Fatalf("second GUI write error = %v, want a control-owner denial", err)
	}
	if err := second.Resize(info.ID, 92, 32); err == nil || !strings.Contains(err.Error(), "session controlled by another GUI") {
		t.Fatalf("second GUI resize error = %v, want a control-owner denial", err)
	}
	if err := first.Detach(info.ID); err != nil {
		t.Fatalf("explicit detach failed: %v", err)
	}
	if err := second.Attach(info.ID); err != nil {
		t.Fatalf("second GUI could not attach after detach: %v", err)
	}
	if err := second.Write(info.ID, "echo attached\n"); err != nil {
		t.Fatalf("new owner write failed: %v", err)
	}
	if err := second.Resize(info.ID, 93, 33); err != nil {
		t.Fatalf("new owner resize failed: %v", err)
	}
	if err := second.Attach(secondInfo.ID); err != nil {
		t.Fatalf("GUI could not attach to a second session: %v", err)
	}

	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	third := dialTestClient(t, listener.Addr().String())
	deadline := time.Now().Add(3 * time.Second)
	for {
		err = third.Attach(info.ID)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "session controlled by another GUI") || time.Now().After(deadline) {
			t.Fatalf("owner was not released after disconnect: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := first.Write(info.ID, "echo old owner\n"); err == nil || !strings.Contains(err.Error(), "session controlled by another GUI") {
		t.Fatalf("previous owner write error = %v, want control-owner denial", err)
	}
	if err := third.Write(info.ID, "echo after disconnect\n"); err != nil {
		t.Fatalf("reconnected GUI write failed: %v", err)
	}
	if err := third.Attach(secondInfo.ID); err != nil {
		t.Fatalf("disconnect did not release the peer's other session lease: %v", err)
	}
	listed, err := first.List()
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, current := range listed {
		for _, previous := range []session.Info{info, secondInfo} {
			if current.ID == previous.ID {
				found[current.ID] = true
				if current.PID != previous.PID || current.Status != "running" {
					t.Fatalf("session did not survive GUI disconnect: before=%+v after=%+v", previous, current)
				}
			}
		}
	}
	if !found[info.ID] || !found[secondInfo.ID] {
		t.Fatal("a session disappeared after GUI disconnect")
	}
}

func waitForPeerPending(t *testing.T, server *Server, client *Client, result <-chan error) bool {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	local := client.conn.LocalAddr().String()
	for time.Now().Before(deadline) {
		server.mu.Lock()
		pending := 0
		for connection := range server.peers {
			if connection.conn.RemoteAddr().String() == local {
				pending = len(connection.pending)
				break
			}
		}
		server.mu.Unlock()
		if pending > 0 {
			return true
		}
		select {
		case <-result:
			return false
		default:
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

func TestAttachValidatesManagedSessionAndAllowsExitedView(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve() }()
	t.Cleanup(func() {
		server.shutdown()
		select {
		case <-serveDone:
		case <-time.After(3 * time.Second):
			t.Error("test daemon did not stop")
		}
	})
	client := dialTestClient(t, listener.Addr().String())
	if err := client.Attach("missing"); !errors.Is(err, session.ErrNotFound) && (err == nil || !strings.Contains(err.Error(), session.ErrNotFound.Error())) {
		t.Fatalf("attach unknown ID error = %v, want session not found", err)
	}
	info, err := client.Create(session.Profiles()[0].ID, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Terminate(info.ID)
		waitForClientSessionExit(t, client, info.ID)
	})
	if err := client.Terminate(info.ID); err != nil {
		t.Fatal(err)
	}
	waitForClientSessionExit(t, client, info.ID)
	if err := client.Attach(info.ID); err != nil {
		t.Fatalf("attach to exited session for read view failed: %v", err)
	}
	second := dialTestClient(t, listener.Addr().String())
	if err := second.Attach(info.ID); err != nil {
		t.Fatalf("second GUI could not attach to exited session for read view: %v", err)
	}
	if _, err := client.Snapshot(info.ID); err != nil {
		t.Fatalf("snapshot of exited session failed: %v", err)
	}
}

func dialTestClient(t *testing.T, address string) *Client {
	t.Helper()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{conn: conn, pending: make(map[uint64]chan message), writeGate: make(chan struct{}, 1), done: make(chan struct{})}
	go client.readLoop()
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func waitForClientSessionExit(t *testing.T, client *Client, id string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		items, err := client.List()
		if err != nil {
			t.Errorf("list sessions during cleanup: %v", err)
			return
		}
		for _, info := range items {
			if info.ID == id && info.Status == "exited" {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("session %s did not exit during cleanup", id)
}
