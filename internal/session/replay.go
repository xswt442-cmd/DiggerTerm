package session

import "encoding/base64"

// Snapshot replays bounded raw bytes; it is not a lossless VT screen snapshot.
// The offset lets the frontend de-duplicate output racing with this call.
func (m *Manager) Snapshot(id string) (Replay, error) {
	s, err := m.get(id)
	if err != nil {
		return Replay{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return Replay{
		Data:      base64.StdEncoding.EncodeToString(s.history),
		Offset:    s.offset,
		Truncated: s.offset > uint64(len(s.history)),
	}, nil
}
