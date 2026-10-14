package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/xswt442-cmd/DiggerTerm/internal/daemon"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

// SessionService is ONLY a Wails-to-IPC proxy. It never owns a PTY.
// Closing the GUI/client connection does not terminate daemon sessions.
type SessionService struct {
	client  *daemon.Client
	initErr error
}

func NewSessionService(app *application.App) *SessionService {
	client, err := daemon.EnsureClient(func(event string, raw json.RawMessage) {
		switch event {
		case "terminal:output":
			var output session.Output
			if json.Unmarshal(raw, &output) == nil {
				app.Event.Emit(event, output)
			}
		case "terminal:exit":
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

func (s *SessionService) Snapshot(id string) (session.Replay, error) {
	c, err := s.backend()
	if err != nil {
		return session.Replay{}, err
	}
	return c.Snapshot(id)
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
