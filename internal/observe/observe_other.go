//go:build !windows

package observe

func platformSnapshot() ([]processRow, bool, error) {
	// Unix process discovery needs a separate same-user implementation;
	// avoid claiming support merely because the app compiles there.
	return nil, false, nil
}
