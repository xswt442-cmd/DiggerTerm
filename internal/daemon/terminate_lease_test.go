package daemon

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

func TestTerminateRespectsAnotherGUIController(t *testing.T) {
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
		// Only the test-created process may be terminated by cleanup.
		_ = first.Detach(info.ID)
		_ = second.Detach(info.ID)
		if items, e := first.List(); e == nil {
			for _, item := range items {
				if item.ID == info.ID && item.Status == "running" {
					_ = first.Terminate(info.ID)
				}
			}
		}
		waitForClientSessionExit(t, first, info.ID)
	})
	if err := first.Attach(info.ID); err != nil {
		t.Fatal(err)
	}
	err = second.Terminate(info.ID)
	if err == nil || !strings.Contains(err.Error(), "controlled by another GUI") {
		t.Fatalf("second GUI termination = %v; want lease denial", err)
	}
	items, err := first.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "running" {
		t.Fatalf("denied termination changed session state: %+v", items)
	}
	if err := first.Detach(info.ID); err != nil {
		t.Fatal(err)
	}
	if err := second.Terminate(info.ID); err != nil {
		t.Fatalf("unowned session should be terminable: %v", err)
	}
	waitForClientSessionExit(t, first, info.ID)
}
