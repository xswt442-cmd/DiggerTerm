package daemon

import (
	"net"
	"sync"
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

func peerClosed(p *peer) bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}
