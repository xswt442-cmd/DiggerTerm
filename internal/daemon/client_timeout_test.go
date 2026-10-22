package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

func TestClientTimeoutDoesNotConsumeLateResponseOrBreakNextRequest(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	client := &Client{conn: clientConn, writeGate: make(chan struct{}, 1),
		pending: make(map[uint64]chan message), done: make(chan struct{})}
	go client.readLoop()
	defer client.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		reader := bufio.NewScanner(serverConn)
		writer := json.NewEncoder(serverConn)
		for i := 0; i < 2 && reader.Scan(); i++ {
			var req request
			if json.Unmarshal(reader.Bytes(), &req) != nil {
				return
			}
			if i == 0 {
				time.Sleep(40 * time.Millisecond)
			}
			if writer.Encode(response(req.ID, "ok", nil)) != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	if err := client.callContext(ctx, request{Method: "ping"}, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("call returned %v; want deadline", err)
	}
	var result string
	if err := client.callContext(context.Background(), request{Method: "ping"}, &result); err != nil {
		t.Fatalf("subsequent call failed: %v", err)
	}
	if result != "ok" {
		t.Fatalf("second response = %q", result)
	}
	<-serverDone
}

func TestClientBlockedIPCWriteHonorsDeadline(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close() // peer intentionally does not read
	client := &Client{conn: clientConn, writeGate: make(chan struct{}, 1),
		pending: make(map[uint64]chan message), done: make(chan struct{})}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := client.callContext(ctx, request{Method: "ping"}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked write returned %v; want deadline", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("blocked write exceeded bounded deadline")
	}
}

func TestClientWaitsForWriteGateOnlyUntilDeadline(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	client := &Client{conn: clientConn, writeGate: make(chan struct{}, 1),
		pending: make(map[uint64]chan message), done: make(chan struct{})}
	defer client.Close()
	client.writeGate <- struct{}{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	err := client.callContext(ctx, request{Method: "ping"}, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("busy write gate returned %v; want deadline", err)
	}
	<-client.writeGate
}
