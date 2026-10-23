package session

import (
	"crypto/rand"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
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
	OutputOffset uint64 `json:"outputOffset"`
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
	sshTarget            string // immutable connection metadata; never derived from renameable display name
	sshPort              int
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
		info := s.info
		info.OutputOffset = s.offset
		result = append(result, info)
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
