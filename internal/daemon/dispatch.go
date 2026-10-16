package daemon

import (
	"errors"
	"fmt"

	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

func isSessionQueuedMethod(method string) bool {
	switch method {
	case "write", "resize", "attach", "detach":
		return true
	default:
		return false
	}
}

func (s *Server) stoppingNow() bool {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	return s.stopping
}

func (s *Server) dispatch(p *peer, req request) message {
	if peerClosed(p) {
		return response(req.ID, nil, errors.New("GUI connection is closed"))
	}
	switch req.Method {
	case "ping":
		return response(req.ID, "diggerterm/v1", nil)
	case "profiles":
		return response(req.ID, session.Profiles(), nil)
	case "list":
		return response(req.ID, s.manager.List(), nil)
	case "sshHosts":
		aliases, err := session.SSHHostAliases()
		return response(req.ID, aliases, err)
	case "create":
		s.lifecycle.Lock()
		defer s.lifecycle.Unlock()
		if s.stopping {
			return response(req.ID, nil, errors.New("daemon is stopping"))
		}
		info, err := s.manager.Create(req.Profile, req.Cols, req.Rows)
		return response(req.ID, info, err)
	case "createLocal":
		s.lifecycle.Lock()
		defer s.lifecycle.Unlock()
		if s.stopping {
			return response(req.ID, nil, errors.New("daemon is stopping"))
		}
		info, err := s.manager.CreateLocal(req.Profile, req.Directory, req.Cols, req.Rows)
		return response(req.ID, info, err)
	case "createSSH":
		s.lifecycle.Lock()
		defer s.lifecycle.Unlock()
		if s.stopping {
			return response(req.ID, nil, errors.New("daemon is stopping"))
		}
		info, err := s.manager.CreateSSH(req.SSHTarget, req.SSHPort, req.Cols, req.Rows)
		return response(req.ID, info, err)
	case "rename":
		info, err := s.manager.Rename(req.SessionID, req.Name)
		if err == nil {
			s.broadcast("terminal:renamed", info)
		}
		return response(req.ID, info, err)
	case "write":
		return response(req.ID, nil, s.withSessionLease(p, req.SessionID, func() error {
			return s.manager.Write(req.SessionID, req.Data)
		}))
	case "resize":
		return response(req.ID, nil, s.withSessionLease(p, req.SessionID, func() error {
			return s.manager.Resize(req.SessionID, req.Cols, req.Rows)
		}))
	case "attach":
		return response(req.ID, nil, s.attachPeer(p, req.SessionID))
	case "detach":
		return response(req.ID, nil, s.detachPeer(p, req.SessionID))
	case "snapshot":
		reply, err := s.manager.Snapshot(req.SessionID)
		return response(req.ID, reply, err)
	case "observe":
		// No target PID parameter and no control operation. Only report
		// current-user metadata, excluding daemon-owned PTY process trees.
		owned := make([]int, 0)
		for _, managed := range s.manager.List() {
			if managed.Status == "running" && managed.PID > 0 {
				owned = append(owned, managed.PID)
			}
		}
		snapshot, err := observe.Scan(owned)
		return response(req.ID, snapshot, err)
	case "terminate":
		return response(req.ID, nil, s.manager.Terminate(req.SessionID))
	case "forget":
		err := s.manager.Forget(req.SessionID)
		if err == nil {
			s.removeSessionLease(req.SessionID)
		}
		return response(req.ID, nil, err)
	case "stop":
		s.lifecycle.Lock()
		defer s.lifecycle.Unlock()
		if s.stopping {
			return response(req.ID, nil, errors.New("daemon is already stopping"))
		}
		for _, item := range s.manager.List() {
			if item.Status == "running" {
				return response(req.ID, nil, errors.New("stop running sessions explicitly before stopping the daemon"))
			}
		}
		s.stopping = true
		return response(req.ID, nil, nil)
	default:
		return response(req.ID, nil, fmt.Errorf("unsupported daemon method %q", req.Method))
	}
}
