package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchDirectory(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	dir := t.TempDir()
	service := &SessionService{}
	for _, tc := range []struct {
		args []string
		want string
		fail bool
	}{
		{[]string{"app"}, "", false},
		{[]string{"app", "--directory", dir}, dir, false},
		{[]string{"app", "--directory", "relative"}, "", true},
		{[]string{"app", "--directory", filepath.Join(dir, "missing")}, "", true},
	} {
		os.Args = tc.args
		got, err := service.LaunchDirectory()
		if (err != nil) != tc.fail || got != tc.want {
			t.Fatalf("args %q: got %q, %v", tc.args, got, err)
		}
	}
}
