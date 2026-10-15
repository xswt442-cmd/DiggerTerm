//go:build !windows

package session

import (
	"errors"
	"io"
	"os"
	"syscall"
)

func expectedPTYReadEnd(err error) bool {
	// Unix PTY masters report EIO when the final slave descriptor closes.
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.EIO)
}
