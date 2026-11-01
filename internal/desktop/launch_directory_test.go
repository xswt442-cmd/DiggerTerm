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

func TestForwardedDirectoryRequestsAreQueuedAndConsumedOnce(t *testing.T) {
	dir := t.TempDir()
	s := &SessionService{}
	for i := 0; i < 2; i++ {
		if err := s.enqueueLaunch([]string{"app", "--directory", dir}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.enqueueLaunch([]string{"app", "--directory", "relative"}); err == nil {
		t.Fatal("accepted relative directory")
	}
	got := s.TakeLaunchDirectories()
	if len(got) != 2 || got[0] != dir || got[1] != dir {
		t.Fatalf("unexpected pending requests: %q", got)
	}
	if got := s.TakeLaunchDirectories(); len(got) != 0 {
		t.Fatalf("requests delivered twice: %q", got)
	}
}
