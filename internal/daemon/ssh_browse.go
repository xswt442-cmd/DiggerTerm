package daemon

import (
	"context"
	"errors"
	"time"
)

// Remote browsing is a bounded, noninteractive side channel. It never sends
// commands to the active PTY or changes the user's shell working directory.
func (s *Server) startBrowseSSH(p *peer, req request) {
	select {
	case s.browseSlots <- struct{}{}:
	default:
		p.publish(response(req.ID, nil, errors.New("too many concurrent remote directory requests")))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	go func() {
		defer cancel()
		defer func() { <-s.browseSlots }()
		go func() {
			select {
			case <-p.done:
				cancel()
			case <-ctx.Done():
			}
		}()
		result, err := s.manager.BrowseSSHDirectories(ctx, req.SessionID, req.Directory)
		p.publish(response(req.ID, result, err))
	}()
}
