package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

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
	ID           string `json:"id"`
	Name         string `json:"name"`
	Profile      string `json:"profile"`
	Status       string `json:"status"`
	PID          int    `json:"pid"`
	Message      string `json:"message"`
	StartedAt    string `json:"startedAt"`
	ExitedAt     string `json:"exitedAt"`
	ExitCode     *int   `json:"exitCode"`
	ExitReason   string `json:"exitReason"`
	StreamClosed bool   `json:"streamClosed"`
	StreamError  string `json:"streamError"`
	Revision     uint64 `json:"revision"`
}

type Output struct {
	ID     string `json:"id"`
	Data   string `json:"data"`   // base64 of raw PTY bytes
	Offset uint64 `json:"offset"` // byte offset at the end of the chunk
}

type Replay struct {
	Data      string `json:"data"`
	Offset    uint64 `json:"offset"`
	Truncated bool   `json:"truncated"`
}

type managed struct {
	mu                   sync.Mutex
	writeMu              sync.Mutex
	info                 Info
	pty                  pty.Pty
	cmd                  *pty.Cmd
	history              []byte
	offset               uint64
	readDone             chan struct{}
	processDone          bool
	terminationRequested bool
}

type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*managed
	order      []string
	nextID     atomic.Uint64
	instanceID string
	output     func(Output)
	exited     func(Info)
	state      func(Info)
}

func NewManager(output func(Output), exited func(Info)) *Manager {
	return &Manager{
		sessions:   make(map[string]*managed),
		instanceID: rand.Text(),
		output:     output,
		exited:     exited,
	}
}

func (m *Manager) SetStateHandler(handler func(Info)) {
	m.mu.Lock()
	m.state = handler
	m.mu.Unlock()
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
	return m.CreateLocal(profile, "", cols, rows)
}

// CreateLocal uses the child's actual OS working directory, never a shell
// command. It is only supported for local shells, not WSL or SSH.
func (m *Manager) CreateLocal(profile, directory string, cols, rows int) (Info, error) {
	spec, ok := profileCommand(profile)
	if !ok {
		return Info{}, fmt.Errorf("unsupported profile: %q", profile)
	}
	if !validSize(cols, rows) {
		return Info{}, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	if directory != "" {
		if profile != "powershell" && profile != "cmd" && profile != "shell" && profile != "bash" {
			return Info{}, fmt.Errorf("custom working directory is not supported for %q", profile)
		}
		if !filepath.IsAbs(directory) {
			return Info{}, errors.New("working directory must be an absolute path")
		}
		stat, err := os.Stat(directory)
		if err != nil {
			return Info{}, fmt.Errorf("working directory: %w", err)
		}
		if !stat.IsDir() {
			return Info{}, errors.New("working directory is not a directory")
		}
		spec.dir = filepath.Clean(directory)
	}
	return m.createSpec(spec, profileName(profile), profile, cols, rows)
}

// CreateSSH delegates authentication, host-key verification and ~/.ssh/config
// handling to the platform OpenSSH client. It owns ONLY the client PTY.
func (m *Manager) CreateSSH(target string, port, cols, rows int) (Info, error) {
	if !validSize(cols, rows) {
		return Info{}, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	spec, err := sshCommand(target, port)
	if err != nil {
		return Info{}, err
	}
	return m.createSpec(spec, "SSH · "+target, "ssh", cols, rows)
}

func (m *Manager) createSpec(spec commandSpec, name, profile string, cols, rows int) (Info, error) {
	terminal, err := pty.New()
	if err != nil {
		return Info{}, fmt.Errorf("create PTY: %w", err)
	}
	if err = terminal.Resize(cols, rows); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("resize PTY: %w", err)
	}
	if spec.dir != "" {
		// go-pty resolves a relative executable against Cmd.Dir, unlike
		// ordinary PATH lookup; make the executable absolute first.
		resolved, resolveErr := exec.LookPath(spec.path)
		if resolveErr != nil {
			_ = terminal.Close()
			return Info{}, fmt.Errorf("find shell executable: %w", resolveErr)
		}
		spec.path = resolved
	}
	cmd := terminal.Command(spec.path, spec.args...)
	cmd.Dir = spec.dir
	if len(spec.env) != 0 {
		cmd.Env = append(os.Environ(), spec.env...)
	}
	if err = cmd.Start(); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("start %s: %w", spec.path, err)
	}

	id := m.instanceID + "-" + strconv.FormatUint(m.nextID.Add(1), 10)
	info := Info{
		ID: id, Name: name, Profile: profile, Status: "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Revision: 1,
	}
	if cmd.Process != nil {
		info.PID = cmd.Process.Pid
	}
	s := &managed{info: info, pty: terminal, cmd: cmd, readDone: make(chan struct{})}
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
	defer close(s.readDone)
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
			s.mu.Lock()
			if !s.info.StreamClosed {
				s.info.StreamClosed = true
				if !expectedPTYReadEnd(err) && !s.processDone {
					s.info.StreamError = "read-failed"
				}
				s.info.Revision++
			}
			info := s.info
			s.mu.Unlock()
			log.Printf("session_stream_closed id=%q pid=%d profile=%q error_category=%q", info.ID, info.PID, info.Profile, info.StreamError)
			m.mu.RLock()
			handler := m.state
			m.mu.RUnlock()
			if handler != nil {
				handler(info)
			}
			return
		}
	}
}

func (m *Manager) waitLoop(s *managed) {
	err := s.cmd.Wait()
	ptyClosed := false
	s.mu.Lock()
	s.processDone = true
	s.mu.Unlock()
	// Allow the PTY reader a short, bounded interval to drain SSH errors
	// (especially host-key rejection) before the ConPTY is closed.
	select {
	case <-s.readDone:
	case <-time.After(300 * time.Millisecond):
		s.writeMu.Lock()
		_ = s.pty.Close()
		s.writeMu.Unlock()
		ptyClosed = true
		select {
		case <-s.readDone:
		case <-time.After(300 * time.Millisecond):
		}
	}
	s.mu.Lock()
	if !s.info.StreamClosed {
		s.info.StreamClosed = true
		s.info.StreamError = "reader-close-timeout"
		s.info.Revision++
	}
	if err != nil {
		s.info.Message = err.Error()
	}
	if state := s.cmd.ProcessState; state != nil {
		code := state.ExitCode()
		s.info.ExitCode = &code
	}
	if s.terminationRequested {
		s.info.ExitReason = "user-terminated"
	} else if s.info.Profile == "ssh" {
		s.info.ExitReason = "ssh-client-exit"
	} else {
		s.info.ExitReason = "process-exit"
	}
	s.info.Status = "exited"
	s.info.ExitedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.info.Revision++
	info := s.info
	s.mu.Unlock()

	// The PTY belongs to this managed session. Its final close must not happen
	// when a tab is switched or hidden; it happens on process exit only.
	if !ptyClosed {
		s.writeMu.Lock()
		_ = s.pty.Close()
		s.writeMu.Unlock()
	}
	exitCode := "unknown"
	if info.ExitCode != nil {
		exitCode = strconv.Itoa(*info.ExitCode)
	}
	log.Printf("session_exit id=%q pid=%d profile=%q exit_code=%s started_at=%q exited_at=%q", info.ID, info.PID, info.Profile, exitCode, info.StartedAt, info.ExitedAt)
	if m.exited != nil {
		m.exited(info)
	}
}

// Rename changes the display name of a daemon-owned session. The name lives
// with the daemon's session metadata, not with GUI tabs or external PIDs.
func (m *Manager) Rename(id, name string) (Info, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return Info{}, errors.New("session name must be 1–64 characters")
	}
	for _, char := range name {
		if unicode.IsControl(char) || unicode.Is(unicode.Cf, char) {
			return Info{}, errors.New("session name cannot contain control or invisible format characters")
		}
	}
	s, err := m.get(id)
	if err != nil {
		return Info{}, err
	}
	s.mu.Lock()
	s.info.Name = name
	s.info.Revision++
	info := s.info
	s.mu.Unlock()
	return info, nil
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
		Data:      base64.StdEncoding.EncodeToString(s.history),
		Offset:    s.offset,
		Truncated: s.offset > uint64(len(s.history)),
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
	if running && s.terminationRequested {
		s.mu.Unlock()
		return errors.New("session termination already requested")
	}
	if running {
		s.terminationRequested = true
	}
	s.mu.Unlock()
	if !running {
		return ErrNotRunning
	}
	if err := s.cmd.Process.Kill(); err != nil {
		s.mu.Lock()
		s.terminationRequested = false
		s.mu.Unlock()
		return err
	}
	return nil
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
