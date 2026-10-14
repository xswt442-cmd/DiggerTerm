package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

type Client struct {
	conn           net.Conn
	writeMu        sync.Mutex
	mu             sync.Mutex
	pending        map[uint64]chan message
	nextID         atomic.Uint64
	done           chan struct{}
	closedByClient atomic.Bool
	events         func(string, json.RawMessage)
}

func Dial(events func(string, json.RawMessage)) (*Client, error) {
	conn, err := dialLocal(400 * time.Millisecond)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn, pending: make(map[uint64]chan message), done: make(chan struct{}), events: events}
	go c.readLoop()
	var ok string
	if err := c.call(request{Method: "ping"}, &ok); err != nil {
		_ = c.Close()
		return nil, err
	}
	if ok != "diggerterm/v1" {
		_ = c.Close()
		return nil, errors.New("unexpected daemon handshake")
	}
	return c, nil
}

func (c *Client) Close() error {
	c.closedByClient.Store(true)
	return c.conn.Close()
}

func (c *Client) readLoop() {
	defer func() {
		close(c.done)
		if !c.closedByClient.Load() && c.events != nil {
			c.events("daemon:disconnected", nil)
		}
	}()
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 4096), maxMessageBytes)
	for scanner.Scan() {
		var msg message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			break
		}
		if msg.Event != "" {
			if c.events != nil {
				c.events(msg.Event, msg.Payload)
			}
			continue
		}
		c.mu.Lock()
		pending := c.pending[msg.ID]
		c.mu.Unlock()
		if pending != nil {
			select {
			case pending <- msg:
			default:
			}
		}
	}
}

func (c *Client) call(req request, out any) error {
	req.ID = c.nextID.Add(1)
	ch := make(chan message, 1)
	c.mu.Lock()
	c.pending[req.ID] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, req.ID)
		c.mu.Unlock()
	}()
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if len(data) > maxMessageBytes-1 {
		return errors.New("IPC request too large")
	}
	c.writeMu.Lock()
	select {
	case <-c.done:
		c.writeMu.Unlock()
		return ErrDisconnected
	default:
	}
	_, err = c.conn.Write(append(data, '\n'))
	c.writeMu.Unlock()
	if err != nil {
		return failed("write IPC", err)
	}
	select {
	case result := <-ch:
		if result.Error != "" {
			return errors.New(result.Error)
		}
		if out != nil {
			if err := json.Unmarshal(result.Result, out); err != nil {
				return fmt.Errorf("decode IPC response: %w", err)
			}
		}
		return nil
	case <-c.done:
		return ErrDisconnected
	}
}

func (c *Client) Profiles() ([]session.Profile, error) {
	var result []session.Profile
	err := c.call(request{Method: "profiles"}, &result)
	return result, err
}
func (c *Client) SSHHosts() ([]string, error) {
	var aliases []string
	err := c.call(request{Method: "sshHosts"}, &aliases)
	return aliases, err
}

func (c *Client) List() ([]session.Info, error) {
	var result []session.Info
	err := c.call(request{Method: "list"}, &result)
	return result, err
}
func (c *Client) Create(profile string, cols, rows int) (session.Info, error) {
	var result session.Info
	err := c.call(request{Method: "create", Profile: profile, Cols: cols, Rows: rows}, &result)
	return result, err
}

// CreateSSH requests a daemon-owned OpenSSH client PTY. All authentication
// and known_hosts prompts appear inside that PTY, never in IPC parameters.
func (c *Client) CreateLocal(profile, directory string, cols, rows int) (session.Info, error) {
	var result session.Info
	err := c.call(request{Method: "createLocal", Profile: profile, Directory: directory, Cols: cols, Rows: rows}, &result)
	return result, err
}

func (c *Client) CreateSSH(target string, port, cols, rows int) (session.Info, error) {
	var result session.Info
	err := c.call(request{Method: "createSSH", SSHTarget: target, SSHPort: port, Cols: cols, Rows: rows}, &result)
	return result, err
}

// Rename only accepts daemon-created session IDs, never observed process IDs.
func (c *Client) Rename(id, name string) (session.Info, error) {
	var info session.Info
	err := c.call(request{Method: "rename", SessionID: id, Name: name}, &info)
	return info, err
}

func (c *Client) Write(id, data string) error {
	return c.call(request{Method: "write", SessionID: id, Data: data}, nil)
}
func (c *Client) Resize(id string, cols, rows int) error {
	return c.call(request{Method: "resize", SessionID: id, Cols: cols, Rows: rows}, nil)
}
func (c *Client) Snapshot(id string) (session.Replay, error) {
	var result session.Replay
	err := c.call(request{Method: "snapshot", SessionID: id}, &result)
	return result, err
}
func (c *Client) Terminate(id string) error {
	return c.call(request{Method: "terminate", SessionID: id}, nil)
}
func (c *Client) Forget(id string) error {
	return c.call(request{Method: "forget", SessionID: id}, nil)
}

// Observe fetches an inventory of process metadata. It has no process
// control or terminal-input endpoint for observed processes.
func (c *Client) Observe() (observe.Snapshot, error) {
	var result observe.Snapshot
	err := c.call(request{Method: "observe"}, &result)
	return result, err
}

// Stop is an explicit action and refuses to stop while owned PTYs are running.
func (c *Client) Stop() error {
	return c.call(request{Method: "stop"}, nil)
}
