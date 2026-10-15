package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/xswt442-cmd/DiggerTerm/internal/daemon"
	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

// SessionService is ONLY a Wails-to-IPC proxy. It never owns a PTY.
// Closing the GUI/client connection does not terminate daemon sessions.
type SessionService struct {
	client        *daemon.Client
	initErr       error
	setUILanguage func(string)
}

func NewSessionService(app *application.App) *SessionService {
	client, err := daemon.EnsureClient(func(event string, raw json.RawMessage) {
		switch event {
		case "terminal:output":
			var output session.Output
			if json.Unmarshal(raw, &output) == nil {
				app.Event.Emit(event, output)
			}
		case "terminal:exit", "terminal:renamed":
			var info session.Info
			if json.Unmarshal(raw, &info) == nil {
				app.Event.Emit(event, info)
			}
		case "daemon:disconnected":
			app.Event.Emit(event, "Background service disconnected; existing session state may be unavailable.")
		}
	})
	return &SessionService{client: client, initErr: err}
}

func (s *SessionService) backend() (*daemon.Client, error) {
	if s.client == nil {
		if s.initErr != nil {
			return nil, fmt.Errorf("connect session daemon: %w", s.initErr)
		}
		return nil, errors.New("session daemon unavailable")
	}
	return s.client, nil
}

func (s *SessionService) close() {
	if s.client != nil {
		_ = s.client.Close()
	}
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
