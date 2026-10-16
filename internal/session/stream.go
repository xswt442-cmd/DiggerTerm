package session

import (
	"encoding/base64"
	"log"
)

func (m *Manager) readLoop(s *managed) {
	defer close(s.readDone)
	buf := make([]byte, maxOutputChunk)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.offset += uint64(n)
			offset := s.offset
			s.history = append(s.history, data...)
			if len(s.history) > maxReplayBytes {
				s.history = append([]byte(nil), s.history[len(s.history)-maxReplayBytes:]...)
			}
			id := s.info.ID
			s.mu.Unlock()
			if m.output != nil {
				m.output(Output{ID: id, Data: base64.StdEncoding.EncodeToString(data), Offset: offset})
			}
		}
		if err != nil {
			s.mu.Lock()
			if !s.info.StreamClosed {
				s.info.StreamClosed = true
				if !expectedPTYReadEnd(err) && !s.processDone {
					s.info.StreamError = "read-failed"
				}
				s.info.Revision++
			}
			info := s.info
			s.mu.Unlock()
			log.Printf("session_stream_closed id=%q pid=%d profile=%q error_category=%q", info.ID, info.PID, info.Profile, info.StreamError)
			m.mu.RLock()
			handler := m.state
			m.mu.RUnlock()
			if handler != nil {
				handler(info)
			}
			return
		}
	}
}
