//go:build windows

package winsecurity

import (
	"context"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	winio "github.com/Microsoft/go-winio"
)

func TestPrivilegeSpecificPipeNameAndDescriptor(t *testing.T) {
	if PipeName("name", false) != "name" || PipeName("name", true) != "name-elevated" {
		t.Fatal("pipe names do not distinguish privileges")
	}
	if strings.Contains(PipeSecurity("S-1-5-21-1", false), ";;;BA)") {
		t.Fatal("ordinary pipe grants access to all administrators")
	}
	if !strings.Contains(PipeSecurity("S-1-5-21-1", true), ";;;BA)") {
		t.Fatal("elevated pipe lacks administrator access")
	}
}

func TestUnprivilegedPipeServerRejected(t *testing.T) {
	elevated, err := Elevated()
	if err != nil {
		t.Fatal(err)
	}
	if elevated {
		t.Skip("requires a non-elevated test process")
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	name := `\\.\pipe\diggerterm-peer-test-` + time.Now().Format("150405.000000000")
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: PipeSecurity(current.Uid, false)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			<-time.After(time.Second)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := VerifyElevatedPeer(conn, true, current.Uid); err == nil {
		t.Fatal("accepted an unprivileged server as elevated")
	}
}

func TestElevatedPipeRejectsStandardClient(t *testing.T) {
	elevated, err := Elevated()
	if err != nil {
		t.Fatal(err)
	}
	if elevated {
		t.Skip("requires a non-elevated test process")
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	name := `\\.\pipe\diggerterm-integrity-test-` + time.Now().Format("150405.000000000")
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: PipeSecurity(current.Uid, true)})
	if err != nil {
		t.Fatalf("create restricted pipe: %v", err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, name)
	if err == nil {
		_ = conn.Close()
		t.Fatal("standard process connected to administrator-only pipe")
	}
}
func TestElevatedPipeRoundTrip(t *testing.T) {
	elevated, err := Elevated()
	if err != nil {
		t.Fatal(err)
	}
	if !elevated {
		if os.Getenv("DIGGERTERM_REQUIRE_ELEVATED_TEST") == "1" {
			t.Fatal("test requires an elevated Windows process")
		}
		t.Skip("requires an elevated Windows process")
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	name := `\\.\pipe\diggerterm-elevated-test-` + time.Now().Format("150405.000000000")
	raw, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: PipeSecurity(current.Uid, true)})
	if err != nil {
		t.Fatal(err)
	}
	listener := ProtectListener(raw, current.Uid, true)
	defer listener.Close()
	accepted := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
		accepted <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := winio.DialPipeContext(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := VerifyElevatedPeer(conn, true, current.Uid); err != nil {
		t.Fatalf("elevated server rejected: %v", err)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("elevated client was not accepted")
	}
}
