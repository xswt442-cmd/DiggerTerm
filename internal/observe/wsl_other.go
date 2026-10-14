//go:build !windows

package observe

func platformWSLSnapshot() ([]Process, bool, string) {
	return nil, false, ""
}
