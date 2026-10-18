//go:build !windows

package observe

import "context"

func platformWSLSnapshot() ([]Process, bool, string) {
	return nil, false, ""
}

func platformWSLSnapshotContext(ctx context.Context) ([]Process, bool, string) {
	processes, truncated, note, _ := platformWSLResultsContext(ctx)
	return processes, truncated, note
}

func platformWSLResultsContext(ctx context.Context) ([]Process, bool, string, []SourceStatus) {
	if err := ctx.Err(); err != nil {
		return nil, false, err.Error(), nil
	}
	processes, truncated, note := platformWSLSnapshot()
	return processes, truncated, note, nil
}
