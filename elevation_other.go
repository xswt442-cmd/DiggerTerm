//go:build !windows

package main

import "errors"

func requestAdminElevation(_ []string) error {
	return errors.New("administrator launch is only supported on Windows")
}
