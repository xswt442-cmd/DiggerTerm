package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/xswt442-cmd/DiggerTerm/internal/daemon"
	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

// SessionService is ONLY a Wails-to-IPC proxy. It never owns a PTY.
// Closing the GUI/client connection does not terminate daemon sessions.
type SessionService struct {
	mu            sync.RWMutex
	reconnectMu   sync.Mutex
	bulkStopMu    sync.Mutex
	bulkStopBusy  bool
	app           *application.App
	client        *daemon.Client
	initErr       error
	generation    uint64
	setUILanguage func(string)
}

func NewSessionService(app *application.App) *SessionService {
	service := &SessionService{app: app, generation: 1}
	client, err := daemon.EnsureClient(service.eventHandler(1))
	service.mu.Lock()
	service.client = client
	service.initErr = err
	service.mu.Unlock()
	return service
}

func (s *SessionService) eventHandler(generation uint64) func(string, json.RawMessage) {
	return func(event string, raw json.RawMessage) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if generation != s.generation || s.app == nil {
			return
		}
		switch event {
		case "terminal:output":
			var output session.Output
			if json.Unmarshal(raw, &output) == nil {
				s.app.Event.Emit(event, output)
			}
		case "terminal:exit", "terminal:renamed", "terminal:state":
			var info session.Info
			if json.Unmarshal(raw, &info) == nil {
				s.app.Event.Emit(event, info)
			}
		case "daemon:disconnected":
			s.app.Event.Emit(event, "Background service disconnected; existing session state may be unavailable.")
		}
	}
}

func (s *SessionService) backend() (*daemon.Client, error) {
	s.mu.RLock()
	client, initErr := s.client, s.initErr
	s.mu.RUnlock()
	if client == nil {
		if initErr != nil {
			return nil, fmt.Errorf("connect session daemon: %w", initErr)
		}
		return nil, errors.New("session daemon unavailable")
	}
	return client, nil
}

func (s *SessionService) close() {
	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()
	s.mu.Lock()
	s.generation++
	client := s.client
	s.client = nil
	s.initErr = errors.New("session service closed")
	s.mu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}

// Reconnect only dials the existing daemon. It never starts or stops the
// daemon, and it returns the sessions currently managed by that process.
func (s *SessionService) Reconnect() ([]session.Info, error) {
	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()

	s.mu.Lock()
	s.generation++
	generation := s.generation
	previous := s.client
	s.client = nil
	s.initErr = errors.New("session daemon reconnecting")
	s.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}

	client, err := daemon.Dial(s.eventHandler(generation))
	var current []session.Info
	if err == nil {
		current, err = client.List()
		if err != nil {
			_ = client.Close()
		}
	}
	if err != nil {
		s.mu.Lock()
		if s.generation == generation {
			s.initErr = err
		}
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Lock()
	if s.generation != generation {
		s.mu.Unlock()
		_ = client.Close()
		return nil, errors.New("session daemon reconnect was superseded")
	}
	s.client = client
	s.initErr = nil
	s.mu.Unlock()

	return current, nil
}

// SetUILanguage changes only the GUI's native menu; it never touches the
// session daemon or any running PTY. The caller sends an effective locale,
// after resolving the UI's "follow system" preference in the WebView.
func (s *SessionService) SetUILanguage(locale string) error {
	if locale != "zh" && locale != "en" {
		return fmt.Errorf("unsupported UI locale %q", locale)
	}
	if s.setUILanguage != nil {
		s.setUILanguage(locale)
	}
	return nil
}

func (s *SessionService) stopDaemon() error {
	client, err := s.backend()
	if err != nil {
		return err
	}
	return client.Stop()
}

type managedSessionStopper interface {
	List() ([]session.Info, error)
	Terminate(string) error
	Stop() error
}

type managedSessionStateChangedError struct {
	currentIDs []string
}

func (e *managedSessionStateChangedError) Error() string {
	return fmt.Sprintf("the managed session list changed; confirm again for %d running session(s)", len(e.currentIDs))
}

const stopAllSessionsTimeout = 15 * time.Second

// stopManagedSessionsAndDaemon operates only on IDs returned by List. It
// revalidates the count after user confirmation and refuses to stop the daemon
// if new running sessions appear while the confirmed set is being terminated.
func stopManagedSessionsAndDaemon(client managedSessionStopper, confirmedIDs []string) error {
	confirmedIDs = append([]string(nil), confirmedIDs...)
	sort.Strings(confirmedIDs)
	managed, err := client.List()
	if err != nil {
		return fmt.Errorf("list managed sessions: %w", err)
	}
	currentIDs := runningManagedSessionIDs(managed)
	if !equalSessionIDs(currentIDs, confirmedIDs) {
		return &managedSessionStateChangedError{currentIDs: currentIDs}
	}
	targets := make(map[string]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		targets[id] = struct{}{}
	}

	terminationErrors := make(map[string]error)
	for id := range targets {
		if err := client.Terminate(id); err != nil {
			terminationErrors[id] = fmt.Errorf("terminate managed session %s: %w", id, err)
		}
	}

	deadline := time.NewTimer(stopAllSessionsTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := client.List()
		if err != nil {
			return fmt.Errorf("wait for managed sessions to exit: %w", err)
		}
		activeTargets := 0
		var unresolvedTerminationErrors []error
		currentIDs := runningManagedSessionIDs(current)
		for _, info := range current {
			if info.Status != "running" {
				continue
			}
			if _, confirmed := targets[info.ID]; !confirmed {
				return &managedSessionStateChangedError{currentIDs: currentIDs}
			}
			activeTargets++
			if err := terminationErrors[info.ID]; err != nil {
				unresolvedTerminationErrors = append(unresolvedTerminationErrors, err)
			}
		}
		if len(unresolvedTerminationErrors) > 0 {
			return errors.Join(unresolvedTerminationErrors...)
		}
		if activeTargets == 0 {
			if err := client.Stop(); err != nil {
				return fmt.Errorf("stop daemon after managed sessions exited: %w", err)
			}
			return nil
		}

		select {
		case <-ticker.C:
		case <-deadline.C:
			return errors.New("timed out waiting for managed DiggerTerm sessions to exit")
		}
	}
}

func runningManagedSessionIDs(items []session.Info) []string {
	ids := make([]string, 0)
	for _, item := range items {
		if item.Status == "running" {
			ids = append(ids, item.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func equalSessionIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func runStopAllAfterConfirmation(confirmed bool, ids []string, stop func([]string) error) error {
	if !confirmed {
		return nil
	}
	return stop(ids)
}

func (s *SessionService) runningManagedSessionIDs() ([]string, error) {
	client, err := s.backend()
	if err != nil {
		return nil, err
	}
	managed, err := client.List()
	if err != nil {
		return nil, fmt.Errorf("list managed sessions: %w", err)
	}
	return runningManagedSessionIDs(managed), nil
}

func (s *SessionService) tryBeginBulkStop() bool {
	s.bulkStopMu.Lock()
	defer s.bulkStopMu.Unlock()
	if s.bulkStopBusy {
		return false
	}
	s.bulkStopBusy = true
	return true
}

func (s *SessionService) finishBulkStop() {
	s.bulkStopMu.Lock()
	s.bulkStopBusy = false
	s.bulkStopMu.Unlock()
}

func (s *SessionService) stopAllSessionsAndDaemon(confirmedIDs []string) error {
	client, err := s.backend()
	if err != nil {
		return err
	}
	return stopManagedSessionsAndDaemon(client, confirmedIDs)
}

func (s *SessionService) Profiles() ([]session.Profile, error) {
	c, err := s.backend()
	if err != nil {
		return nil, err
	}
	return c.Profiles()
}

// SSHHosts exposes only simple, explicit Host aliases; never SSH credentials.
func (s *SessionService) SSHHosts() ([]string, error) {
	c, err := s.backend()
	if err != nil {
		return nil, err
	}
	return c.SSHHosts()
}

func (s *SessionService) List() ([]session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return nil, err
	}
	return c.List()
}

func (s *SessionService) Create(profile string, cols, rows int) (session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return session.Info{}, err
	}
	return c.Create(profile, cols, rows)
}

// CreateSSH is a separate operation from Create so untrusted host strings
// cannot masquerade as local session profiles.
func (s *SessionService) CreateLocal(profile, directory string, cols, rows int) (session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return session.Info{}, err
	}
	return c.CreateLocal(profile, directory, cols, rows)
}

func (s *SessionService) CreateSSH(target string, port, cols, rows int) (session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return session.Info{}, err
	}
	return c.CreateSSH(target, port, cols, rows)
}

func (s *SessionService) CreateSSHWithOptions(target string, port int, options session.SSHOptions, cols, rows int) (session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return session.Info{}, err
	}
	return c.CreateSSHWithOptions(target, port, options, cols, rows)
}

func (s *SessionService) Rename(id, name string) (session.Info, error) {
	c, err := s.backend()
	if err != nil {
		return session.Info{}, err
	}
	return c.Rename(id, name)
}

func (s *SessionService) Write(id, data string) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Write(id, data)
}

func (s *SessionService) Resize(id string, cols, rows int) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Resize(id, cols, rows)
}

func (s *SessionService) Attach(id string) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Attach(id)
}

func (s *SessionService) Detach(id string) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Detach(id)
}

func (s *SessionService) Snapshot(id string) (session.Replay, error) {
	c, err := s.backend()
	if err != nil {
		return session.Replay{}, err
	}
	return c.Snapshot(id)
}

// Observe exposes read-only external process metadata. The Wails frontend
// cannot send these PID identifiers to any managed-session control method.
func (s *SessionService) Observe() (observe.Snapshot, error) {
	c, err := s.backend()
	if err != nil {
		return observe.Snapshot{}, err
	}
	return c.Observe()
}

func (s *SessionService) ObserveCapabilities() (daemon.Capabilities, error) {
	client, err := s.backend()
	if err != nil {
		return daemon.Capabilities{}, err
	}
	return client.Capabilities()
}

func (s *SessionService) Terminate(id string) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Terminate(id)
}

func (s *SessionService) Forget(id string) error {
	c, err := s.backend()
	if err != nil {
		return err
	}
	return c.Forget(id)
}
