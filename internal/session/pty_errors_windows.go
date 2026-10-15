//go:build windows

package session

import (
	"errors"
	"io"
	"os"
	"syscall"
)

func expectedPTYReadEnd(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, syscall.ERROR_BROKEN_PIPE)
}
