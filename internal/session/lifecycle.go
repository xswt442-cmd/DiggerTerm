package session

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"
)

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
