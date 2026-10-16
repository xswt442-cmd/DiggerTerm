package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

func TestSlowObserveDoesNotBlockPeerRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseScan := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseScan)
	address := startObserveTestServer(t, func(ctx context.Context, _ []int) (observe.Snapshot, error) {
		close(started)
		select {
		case <-release:
			return observe.Snapshot{Supported: true, Processes: []observe.Process{}}, nil
		case <-ctx.Done():
			return observe.Snapshot{}, ctx.Err()
		}
	})
	first := dialTestClient(t, address)
	second := dialTestClient(t, address)

	observeDone := make(chan error, 1)
	go func() {
		_, err := first.Observe()
		observeDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("observe scan did not start")
	}

	assertReturnsQuickly(t, "same-peer List", func() error {
		_, err := first.List()
		return err
	})
	assertReturnsQuickly(t, "same-peer Create", func() error {
		info, err := first.Create(session.Profiles()[0].ID, 80, 24)
		if err == nil {
			t.Cleanup(func() {
				_ = first.Terminate(info.ID)
				waitForClientSessionExit(t, first, info.ID)
			})
			return nil
		}
		return err
	})
	assertReturnsQuickly(t, "other-peer List", func() error {
		_, err := second.List()
		return err
	})
	if _, err := second.Observe(); err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("concurrent Observe error = %v, want a fast in-progress error", err)
	}

	releaseScan()
	select {
	case err := <-observeDone:
		if err != nil {
			t.Fatalf("Observe after releasing the isolated scan: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Observe response did not arrive after releasing the isolated scan")
	}
}

func TestObserveDisconnectCancelsScanAndReleasesSlot(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan error, 1)
	var calls int
	var callsMu sync.Mutex
	address := startObserveTestServer(t, func(ctx context.Context, _ []int) (observe.Snapshot, error) {
		callsMu.Lock()
		calls++
		call := calls
		callsMu.Unlock()
		if call > 1 {
			return observe.Snapshot{Supported: true, Processes: []observe.Process{}}, nil
		}
		close(started)
		<-ctx.Done()
		canceled <- ctx.Err()
		return observe.Snapshot{}, ctx.Err()
	})
	first := dialTestClient(t, address)
	observeDone := make(chan error, 1)
	go func() {
		_, err := first.Observe()
		observeDone <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("observe scan did not start")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-canceled:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("scan cancellation error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("peer disconnect did not cancel the isolated scan")
	}

	second := dialTestClient(t, address)
	deadline := time.Now().Add(time.Second)
	for {
		_, err := second.Observe()
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "already in progress") || time.Now().After(deadline) {
			t.Fatalf("Observe did not recover after canceled scan: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestObserveErrorIsReturnedOverIPC(t *testing.T) {
	address := startObserveTestServer(t, func(context.Context, []int) (observe.Snapshot, error) {
		return observe.Snapshot{}, errors.New("isolated scan failure")
	})
	client := dialTestClient(t, address)
	if _, err := client.Observe(); err == nil || err.Error() != "isolated scan failure" {
		t.Fatalf("Observe error = %v, want isolated scan failure", err)
	}
}

func TestQueuedWriteAndResizeRepliesRemainFIFO(t *testing.T) {
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
	creator := dialTestClient(t, listener.Addr().String())
	info, err := creator.Create("shell", 80, 24)
	if err != nil {
		t.Skipf("create isolated test shell: %v", err)
	}
	t.Cleanup(func() {
		_ = creator.Terminate(info.ID)
		waitForClientSessionExit(t, creator, info.ID)
	})

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	lease := server.sessionLease(info.ID)
	lease.mu.Lock()
	write, _ := json.Marshal(request{ID: 1, Method: "write", SessionID: info.ID, Data: "\n"})
	resize, _ := json.Marshal(request{ID: 2, Method: "resize", SessionID: info.ID, Cols: 90, Rows: 28})
	if _, err := conn.Write(append(append(write, '\n'), append(resize, '\n')...)); err != nil {
		lease.mu.Unlock()
		t.Fatal(err)
	}
	if !waitForPeerPendingCount(t, server, conn, 2) {
		lease.mu.Unlock()
		t.Fatal("write and resize did not enter the bounded per-session queue")
	}
	lease.mu.Unlock()

	reader := bufio.NewReader(conn)
	for wantID := uint64(1); wantID <= 2; {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read queued response %d: %v", wantID, err)
		}
		var reply message
		if err := json.Unmarshal(line, &reply); err != nil {
			t.Fatalf("decode queued response: %v", err)
		}
		if reply.Event != "" {
			continue
		}
		if reply.ID != wantID {
			t.Fatalf("queued response order = %d, want %d", reply.ID, wantID)
		}
		if reply.Error != "" {
			t.Fatalf("queued request %d failed: %s", wantID, reply.Error)
		}
		wantID++
	}
}

func startObserveTestServer(t *testing.T, scanner func(context.Context, []int) (observe.Snapshot, error)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(listener)
	server.observeScan = scanner
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve() }()
	t.Cleanup(func() {
		server.shutdown()
		select {
		case <-serveDone:
		case <-time.After(3 * time.Second):
			t.Error("isolated test daemon did not stop")
		}
	})
	return listener.Addr().String()
}

func assertReturnsQuickly(t *testing.T, label string, action func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- action() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s waited behind the isolated slow scan", label)
	}
}

func waitForPeerPendingCount(t *testing.T, server *Server, conn net.Conn, want int) bool {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		server.mu.Lock()
		pending := 0
		for p := range server.peers {
			if p.conn.RemoteAddr().String() == conn.LocalAddr().String() {
				pending = len(p.pending)
				break
			}
		}
		server.mu.Unlock()
		if pending == want {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}
