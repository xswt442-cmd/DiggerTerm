package daemon

import (
	"errors"
	"sync"

	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

type sessionLease struct {
	mu    sync.Mutex
	owner *peer
}

func (s *Server) attachPeer(p *peer, id string) error {
	if _, err := s.managedSession(id); err != nil {
		return err
	}
	lease := s.sessionLease(id)
	lease.mu.Lock()
	defer lease.mu.Unlock()
	info, err := s.managedSession(id)
	if err != nil {
		s.pruneSessionLeaseLocked(id, lease)
		return err
	}
	if info.Status != "running" {
		lease.owner = nil
		return nil
	}
	return s.claimLocked(lease, p)
}

func (s *Server) detachPeer(p *peer, id string) error {
	lease := s.findSessionLease(id)
	if lease == nil {
		return nil
	}
	lease.mu.Lock()
	if lease.owner == p {
		lease.owner = nil
	}
	lease.mu.Unlock()
	return nil
}

// A session lease stays locked through the manager operation, so peers cannot
// race ownership of one PTY while unrelated sessions remain independent.
func (s *Server) withSessionLease(p *peer, id string, operation func() error) error {
	if _, err := s.managedSession(id); err != nil {
		return err
	}
	lease := s.sessionLease(id)
	lease.mu.Lock()
	defer lease.mu.Unlock()
	info, err := s.managedSession(id)
	if err != nil {
		s.pruneSessionLeaseLocked(id, lease)
		return err
	}
	if info.Status != "running" {
		lease.owner = nil
		return operation()
	}
	if err := s.claimLocked(lease, p); err != nil {
		return err
	}
	err = operation()
	if updated, lookupErr := s.managedSession(id); lookupErr != nil || updated.Status != "running" || peerClosed(p) {
		if lease.owner == p {
			lease.owner = nil
		}
	}
	return err
}

func (s *Server) claimLocked(lease *sessionLease, p *peer) error {
	if peerClosed(p) {
		return errors.New("GUI connection is closed")
	}
	owner := lease.owner
	if owner != nil && owner != p && !peerClosed(owner) {
		return errors.New("session controlled by another GUI")
	}
	lease.owner = p
	return nil
}

func (s *Server) managedSession(id string) (session.Info, error) {
	for _, info := range s.manager.List() {
		if info.ID == id {
			return info, nil
		}
	}
	return session.Info{}, session.ErrNotFound
}

func (s *Server) releasePeer(p *peer) {
	s.attachmentMu.Lock()
	leases := make([]*sessionLease, 0, len(s.leases))
	for _, lease := range s.leases {
		leases = append(leases, lease)
	}
	s.attachmentMu.Unlock()
	for _, lease := range leases {
		lease.mu.Lock()
		if lease.owner == p {
			lease.owner = nil
		}
		lease.mu.Unlock()
	}
}

func (s *Server) sessionLease(id string) *sessionLease {
	s.attachmentMu.Lock()
	lease := s.leases[id]
	if lease == nil {
		lease = &sessionLease{}
		s.leases[id] = lease
	}
	s.attachmentMu.Unlock()
	return lease
}

func (s *Server) findSessionLease(id string) *sessionLease {
	s.attachmentMu.Lock()
	lease := s.leases[id]
	s.attachmentMu.Unlock()
	return lease
}

func (s *Server) removeSessionLease(id string) {
	lease := s.findSessionLease(id)
	if lease == nil {
		return
	}
	lease.mu.Lock()
	s.attachmentMu.Lock()
	if s.leases[id] == lease {
		lease.owner = nil
		delete(s.leases, id)
	}
	s.attachmentMu.Unlock()
	lease.mu.Unlock()
}

// pruneSessionLeaseLocked is called with lease.mu held after a stale lookup.
// The pointer comparison keeps it from deleting a newer lease entry.
func (s *Server) pruneSessionLeaseLocked(id string, lease *sessionLease) {
	if lease.owner != nil {
		return
	}
	s.attachmentMu.Lock()
	if s.leases[id] == lease {
		delete(s.leases, id)
	}
	s.attachmentMu.Unlock()
}
