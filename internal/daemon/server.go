package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

const maxPeerPendingRequests = 256

type queuedRequest struct {
	req request
}

type sessionRequestQueue struct {
	requests []queuedRequest
}

type peer struct {
	conn         net.Conn
	out          chan message
	done         chan struct{}
	once         sync.Once
	queueMu      sync.Mutex
	pending      chan struct{}
	sessionQueue map[string]*sessionRequestQueue
}

type sessionLease struct {
	mu    sync.Mutex
	owner *peer
}

func (p *peer) close() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
		p.discardQueuedRequests()
	})
}

func (p *peer) enqueueSessionRequest(server *Server, req request) bool {
	select {
	case <-p.done:
		return false
	default:
	}
	select {
	case p.pending <- struct{}{}:
	default:
		p.close()
		return false
	}
	p.queueMu.Lock()
	if peerClosed(p) {
		p.queueMu.Unlock()
		<-p.pending
		return false
	}
	if p.sessionQueue == nil {
		p.sessionQueue = make(map[string]*sessionRequestQueue)
	}
	queue := p.sessionQueue[req.SessionID]
	if queue == nil {
		queue = &sessionRequestQueue{}
		p.sessionQueue[req.SessionID] = queue
		queue.requests = append(queue.requests, queuedRequest{req: req})
		go p.runSessionQueue(server, req.SessionID, queue)
	} else {
		queue.requests = append(queue.requests, queuedRequest{req: req})
	}
	p.queueMu.Unlock()
	return true
}

func (p *peer) runSessionQueue(server *Server, sessionID string, queue *sessionRequestQueue) {
	for {
		p.queueMu.Lock()
		if peerClosed(p) {
			p.queueMu.Unlock()
			return
		}
		if len(queue.requests) == 0 {
			if p.sessionQueue[sessionID] == queue {
				delete(p.sessionQueue, sessionID)
			}
			p.queueMu.Unlock()
			return
		}
		queued := queue.requests[0]
		queue.requests[0] = queuedRequest{}
		queue.requests = queue.requests[1:]
		p.queueMu.Unlock()

		if !peerClosed(p) {
			p.publish(server.dispatch(p, queued.req))
		}
		<-p.pending
	}
}

func (p *peer) discardQueuedRequests() {
	p.queueMu.Lock()
	count := 0
	for _, queue := range p.sessionQueue {
		count += len(queue.requests)
		queue.requests = nil
	}
	p.sessionQueue = nil
	p.queueMu.Unlock()
	for range count {
		<-p.pending
	}
}

// A slow GUI never blocks PTY reads. It can reconnect and retrieve a bounded
// snapshot; dropping this client never affects the session process.
func (p *peer) publish(m message) {
	select {
	case <-p.done:
	case p.out <- m:
	default:
		p.close()
	}
}

type Server struct {
	listener     net.Listener
	manager      *session.Manager
	mu           sync.Mutex
	peers        map[*peer]struct{}
	closing      bool
	lifecycle    sync.Mutex // serialize Create and Stop, not terminal I/O
	stopping     bool
	attachmentMu sync.Mutex
	leases       map[string]*sessionLease
}

func NewServer(listener net.Listener) *Server {
	s := &Server{listener: listener, peers: make(map[*peer]struct{}), leases: make(map[string]*sessionLease)}
	s.manager = session.NewManager(
		func(output session.Output) { s.broadcast("terminal:output", output) },
		func(info session.Info) { s.broadcast("terminal:exit", info) },
	)
	s.manager.SetStateHandler(func(info session.Info) { s.broadcast("terminal:state", info) })
	return s
}

func (s *Server) broadcast(event string, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	m := message{Event: event, Payload: payload}
	s.mu.Lock()
	defer s.mu.Unlock()
	for p := range s.peers {
		p.publish(m)
	}
}

func (s *Server) shutdown() {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return
	}
	s.closing = true
	peers := make([]*peer, 0, len(s.peers))
	for p := range s.peers {
		peers = append(peers, p)
	}
	s.mu.Unlock()
	_ = s.listener.Close()
	for _, p := range peers {
		p.close()
	}
}

func (s *Server) Serve() error {
	defer s.shutdown()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			closing := s.closing
			s.mu.Unlock()
			if closing || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept daemon IPC: %w", err)
		}
		p := &peer{
			conn: conn, out: make(chan message, 128), done: make(chan struct{}),
			pending:      make(chan struct{}, maxPeerPendingRequests),
			sessionQueue: make(map[string]*sessionRequestQueue),
		}
		s.mu.Lock()
		if s.closing {
			s.mu.Unlock()
			p.close()
			return nil
		}
		s.peers[p] = struct{}{}
		s.mu.Unlock()
		go s.servePeer(p)
	}
}

func (s *Server) servePeer(p *peer) {
	defer func() {
		p.close()
		s.releasePeer(p)
		s.mu.Lock()
		delete(s.peers, p)
		s.mu.Unlock()
	}()

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		writer := bufio.NewWriter(p.conn)
		for {
			select {
			case <-p.done:
				return
			case msg := <-p.out:
				data, err := json.Marshal(msg)
				if err != nil {
					p.close()
					return
				}
				if _, err = writer.Write(append(data, '\n')); err != nil {
					p.close()
					return
				}
				if err = writer.Flush(); err != nil {
					p.close()
					return
				}
				if msg.stopAfterWrite {
					s.shutdown()
					return
				}
			}
		}
	}()

	scanner := bufio.NewScanner(p.conn)
	scanner.Buffer(make([]byte, 4096), maxMessageBytes)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil || req.ID == 0 {
			break
		}
		if peerClosed(p) {
			break
		}
		if isSessionQueuedMethod(req.Method) {
			if !p.enqueueSessionRequest(s, req) {
				break
			}
			continue
		}
		reply := s.dispatch(p, req)
		if req.Method == "stop" && reply.Error == "" {
			reply.stopAfterWrite = true
		}
		p.publish(reply)
		if reply.stopAfterWrite {
			// The writer closes the service AFTER flushing this response.
			// Continue reading until the writer closes the pipe.
			break
		}
	}
	if !s.stoppingNow() {
		p.close()
	}
	<-writerDone
}

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

func peerClosed(p *peer) bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

func Run() error {
	if err := prepareDaemonConsoleInterrupts(); err != nil {
		return failed("prepare Windows console Ctrl+C", err)
	}
	listener, err := listenLocal()
	if err != nil {
		return failed("listen local IPC", err)
	}
	return NewServer(listener).Serve()
}
