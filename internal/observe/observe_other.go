//go:build !windows

package observe

import "context"

func platformSnapshot(ctx context.Context) ([]processRow, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	// Unix process discovery needs a separate same-user implementation;
	// avoid claiming support merely because the app compiles there.
	return nil, false, nil
}
