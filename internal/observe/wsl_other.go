//go:build !windows

package observe

import "context"

func platformWSLSnapshot() ([]Process, bool, string) {
	return nil, false, ""
}

func platformWSLSnapshotContext(ctx context.Context) ([]Process, bool, string) {
	if err := ctx.Err(); err != nil {
		return nil, false, err.Error()
	}
	return platformWSLSnapshot()
}
