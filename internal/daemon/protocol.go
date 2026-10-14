package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const maxMessageBytes = 1024 * 1024

// An optional scoped endpoint is useful for concurrent dev builds and tests.
// It is inherited by the daemon and does not change the same-user ACL.
func endpointSuffix() (string, error) {
	raw := os.Getenv("DIGGERTERM_IPC_NAMESPACE")
	if len(raw) > 32 {
		return "", errors.New("IPC namespace exceeds 32 bytes")
	}
	for _, ch := range raw {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return "", errors.New("IPC namespace contains invalid characters")
		}
	}
	if raw == "" {
		return "", nil
	}
	return "-" + raw, nil
}

// Each IPC connection carries newline-delimited JSON requests, replies and events.
// The daemon is the only owner of PTYs, including when all GUI clients disconnect.
type request struct {
	ID        uint64 `json:"id"`
	Method    string `json:"method"`
	SessionID string `json:"sessionId,omitempty"`
	Name      string `json:"name,omitempty"`
	Profile   string `json:"profile,omitempty"`
	SSHTarget string `json:"sshTarget,omitempty"`
	SSHPort   int    `json:"sshPort,omitempty"`
	Data      string `json:"data,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
}

type message struct {
	ID             uint64          `json:"id,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
	Event          string          `json:"event,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	stopAfterWrite bool            `json:"-"`
}

func response(id uint64, value any, err error) message {
	m := message{ID: id}
	if err != nil {
		m.Error = err.Error()
		return m
	}
	body, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		m.Error = marshalErr.Error()
	} else {
		m.Result = body
	}
	return m
}

var ErrDisconnected = errors.New("DiggerTerm session daemon disconnected")

func failed(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
