package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"

	pty "github.com/aymanbagabas/go-pty"
)

const (
	maxReplayBytes = 256 * 1024
	maxOutputChunk = 16 * 1024
)

var (
	ErrNotFound   = errors.New("session not found")
	ErrNotRunning = errors.New("session is not running")
	ErrRunning    = errors.New("cannot forget a running session")
)

type Info struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Profile string `json:"profile"`
	Status  string `json:"status"`
	PID     int    `json:"pid"`
	Message string `json:"message"`
}

type Output struct {
	ID     string `json:"id"`
	Data   string `json:"data"`   // base64 of raw PTY bytes
	Offset uint64 `json:"offset"` // byte offset at the end of the chunk
}

type Replay struct {
	Data   string `json:"data"`
	Offset uint64 `json:"offset"`
}

type managed struct {
	mu      sync.Mutex
	writeMu sync.Mutex
	info    Info
	pty     pty.Pty
	cmd     *pty.Cmd
	history []byte
	offset  uint64
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*managed
	order    []string
	nextID   atomic.Uint64
	output   func(Output)
	exited   func(Info)
}

func NewManager(output func(Output), exited func(Info)) *Manager {
	return &Manager{
		sessions: make(map[string]*managed),
		output:   output,
		exited:   exited,
	}
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Info, 0, len(m.order))
	for _, id := range m.order {
		s := m.sessions[id]
		s.mu.Lock()
		result = append(result, s.info)
		s.mu.Unlock()
	}
	return result
}

func (m *Manager) get(id string) (*managed, error) {
	m.mu.RLock()
	s := m.sessions[id]
	m.mu.RUnlock()
	if s == nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func validSize(cols, rows int) bool {
	return cols >= 1 && cols <= 500 && rows >= 1 && rows <= 500
}

func (m *Manager) Create(profile string, cols, rows int) (Info, error) {
	spec, ok := profileCommand(profile)
	if !ok {
		return Info{}, fmt.Errorf("unsupported profile: %q", profile)
	}
	if !validSize(cols, rows) {
		return Info{}, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}

	terminal, err := pty.New()
	if err != nil {
		return Info{}, fmt.Errorf("create PTY: %w", err)
	}
	if err = terminal.Resize(cols, rows); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("resize PTY: %w", err)
	}
	cmd := terminal.Command(spec.path, spec.args...)
	if err = cmd.Start(); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("start %s: %w", spec.path, err)
	}

	id := strconv.FormatUint(m.nextID.Add(1), 10)
	info := Info{ID: id, Name: profileName(profile), Profile: profile, Status: "running"}
	if cmd.Process != nil {
		info.PID = cmd.Process.Pid
	}
	s := &managed{info: info, pty: terminal, cmd: cmd}
	m.mu.Lock()
	m.sessions[id] = s
	m.order = append(m.order, id)
	m.mu.Unlock()

	go m.readLoop(s)
	go m.waitLoop(s)
	return info, nil
}

func profileName(id string) string {
	for _, p := range Profiles() {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}

func (m *Manager) readLoop(s *managed) {
	buf := make([]byte, maxOutputChunk)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.offset += uint64(n)
			offset := s.offset
			s.history = append(s.history, data...)
			if len(s.history) > maxReplayBytes {
				s.history = append([]byte(nil), s.history[len(s.history)-maxReplayBytes:]...)
			}
			id := s.info.ID
			s.mu.Unlock()
			if m.output != nil {
				m.output(Output{ID: id, Data: base64.StdEncoding.EncodeToString(data), Offset: offset})
			}
		}
		if err != nil {
			return
		}
	}
}

func (m *Manager) waitLoop(s *managed) {
	err := s.cmd.Wait()
	s.mu.Lock()
	s.info.Status = "exited"
	if err != nil {
		s.info.Message = err.Error()
	}
	info := s.info
	s.mu.Unlock()

	// The PTY belongs to this managed session. Its final close must not happen
	// when a tab is switched or hidden; it happens on process exit only.
	s.writeMu.Lock()
	_ = s.pty.Close()
	s.writeMu.Unlock()
	if m.exited != nil {
		m.exited(info)
	}
}

func (m *Manager) Write(id, text string) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	running := s.info.Status == "running"
	s.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	_, err = io.WriteString(s.pty, text)
	return err
}

func (m *Manager) Resize(id string, cols, rows int) error {
	if !validSize(cols, rows) {
		return fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	running := s.info.Status == "running"
	s.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	return s.pty.Resize(cols, rows)
}

// Snapshot replays bounded raw bytes; it is not a lossless VT screen snapshot.
// The offset lets the frontend de-duplicate output racing with this call.
func (m *Manager) Snapshot(id string) (Replay, error) {
	s, err := m.get(id)
	if err != nil {
		return Replay{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return Replay{
		Data:   base64.StdEncoding.EncodeToString(s.history),
		Offset: s.offset,
	}, nil
}

// Terminate only accepts session IDs created by this manager. External
// observed processes are never present in this map.
func (m *Manager) Terminate(id string) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	running := s.info.Status == "running"
	s.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	return s.cmd.Process.Kill()
}

// Forget deliberately refuses running sessions: removing their owner without
// ending them requires explicit detach semantics that do not yet exist.
func (m *Manager) Forget(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil {
		return ErrNotFound
	}
	s.mu.Lock()
	running := s.info.Status == "running"
	s.mu.Unlock()
	if running {
		return ErrRunning
	}
	delete(m.sessions, id)
	for i, item := range m.order {
		if item == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return nil
}
