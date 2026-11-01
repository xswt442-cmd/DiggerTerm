//go:build windows

package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"time"

	winio "github.com/Microsoft/go-winio"
	"github.com/xswt442-cmd/DiggerTerm/internal/winsecurity"
)

type guiLaunchBroker struct {
	listener net.Listener
	launches chan []string
}

func acquireGUIInstance(args []string) (*guiLaunchBroker, bool, error) {
	current, err := user.Current()
	if err != nil || current.Uid == "" {
		return nil, false, errors.New("cannot identify current user for GUI instance")
	}
	namespace := os.Getenv("DIGGERTERM_IPC_NAMESPACE")
	if len(namespace) > 32 {
		return nil, false, errors.New("invalid GUI namespace")
	}
	for _, c := range namespace {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return nil, false, errors.New("invalid GUI namespace")
		}
	}
	elevated, err := winsecurity.Elevated()
	if err != nil {
		return nil, false, err
	}
	name := winsecurity.PipeName(`\\.\pipe\diggerterm-gui-`+current.Uid+"-"+namespace, elevated)
	return acquireGUIInstanceAt(name, current.Uid, args)
}

func acquireGUIInstanceAt(name, sid string, args []string) (*guiLaunchBroker, bool, error) {
	elevated, err := winsecurity.Elevated()
	if err != nil {
		return nil, false, err
	}
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: winsecurity.PipeSecurity(sid, elevated)})
	if err == nil {
		b := &guiLaunchBroker{listener: winsecurity.ProtectListener(listener, sid, elevated), launches: make(chan []string, 64)}
		go b.serve()
		return b, true, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, dialErr := winio.DialPipeContext(ctx, name)
	if dialErr != nil {
		return nil, false, fmt.Errorf("connect existing GUI: %w", dialErr)
	}
	defer conn.Close()
	if elevated {
		if err := winsecurity.VerifyElevatedPeer(conn, true, sid); err != nil {
			return nil, false, err
		}
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := json.NewEncoder(conn).Encode(args); err != nil {
		return nil, false, fmt.Errorf("forward GUI launch: %w", err)
	}
	var ack [1]byte
	if _, err := io.ReadFull(conn, ack[:]); err != nil || ack[0] != 1 {
		return nil, false, errors.New("GUI launch outcome unknown")
	}
	return nil, false, nil
}

func (b *guiLaunchBroker) serve() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			return
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		var args []string
		if json.NewDecoder(io.LimitReader(conn, 16384)).Decode(&args) == nil {
			select {
			case b.launches <- args:
				_, _ = conn.Write([]byte{1})
			default:
			}
		}
		_ = conn.Close()
	}
}

func (b *guiLaunchBroker) close() { _ = b.listener.Close() }
