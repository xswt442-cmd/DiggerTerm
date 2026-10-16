package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/xswt442-cmd/DiggerTerm/internal/observe"
	"github.com/xswt442-cmd/DiggerTerm/internal/session"
)

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
	observeSlots chan struct{}
	observeScan  func(context.Context, []int) (observe.Snapshot, error)
}

func NewServer(listener net.Listener) *Server {
	s := &Server{
		listener: listener, peers: make(map[*peer]struct{}), leases: make(map[string]*sessionLease),
		observeSlots: make(chan struct{}, 1), observeScan: observe.ScanContext,
	}
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
		if req.Method == "observe" {
			s.startObserve(p, req)
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
