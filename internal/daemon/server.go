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

type peer struct {
	conn net.Conn
	out  chan message
	done chan struct{}
	once sync.Once
}

func (p *peer) close() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
	})
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
	listener  net.Listener
	manager   *session.Manager
	mu        sync.Mutex
	peers     map[*peer]struct{}
	closing   bool
	lifecycle sync.Mutex // serialize Create and Stop, not terminal I/O
	stopping  bool
}

func NewServer(listener net.Listener) *Server {
	s := &Server{listener: listener, peers: make(map[*peer]struct{})}
	s.manager = session.NewManager(
		func(output session.Output) { s.broadcast("terminal:output", output) },
		func(info session.Info) { s.broadcast("terminal:exit", info) },
	)
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
		p := &peer{conn: conn, out: make(chan message, 128), done: make(chan struct{})}
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
		reply := s.dispatch(req)
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

func (s *Server) stoppingNow() bool {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	return s.stopping
}

func (s *Server) dispatch(req request) message {
	switch req.Method {
	case "ping":
		return response(req.ID, "diggerterm/v1", nil)
	case "profiles":
		return response(req.ID, session.Profiles(), nil)
	case "list":
		return response(req.ID, s.manager.List(), nil)
	case "create":
		s.lifecycle.Lock()
		defer s.lifecycle.Unlock()
		if s.stopping {
			return response(req.ID, nil, errors.New("daemon is stopping"))
		}
		info, err := s.manager.Create(req.Profile, req.Cols, req.Rows)
		return response(req.ID, info, err)
	case "write":
		return response(req.ID, nil, s.manager.Write(req.SessionID, req.Data))
	case "resize":
		return response(req.ID, nil, s.manager.Resize(req.SessionID, req.Cols, req.Rows))
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
		return response(req.ID, nil, s.manager.Forget(req.SessionID))
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

func Run() error {
	listener, err := listenLocal()
	if err != nil {
		return failed("listen local IPC", err)
	}
	return NewServer(listener).Serve()
}
