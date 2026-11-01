//go:build windows

package desktop

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"testing"
	"time"
)

func TestGUIInstanceForwardsLaunchWithoutCreatingSecondOwner(t *testing.T) {
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	name := `\\.\pipe\diggerterm-gui-test-` + time.Now().Format("150405.000000000")
	first, primary, err := acquireGUIInstanceAt(name, current.Uid, nil)
	if err != nil || !primary {
		t.Fatalf("first instance: primary=%v err=%v", primary, err)
	}
	defer first.close()
	_, primary, err = acquireGUIInstanceAt(name, current.Uid, []string{"app.exe", "--directory", `C:\folder with spaces`})
	if err != nil || primary {
		t.Fatalf("second instance: primary=%v err=%v", primary, err)
	}
	select {
	case args := <-first.launches:
		if len(args) != 3 || args[2] != `C:\folder with spaces` {
			t.Fatalf("bad forwarded args: %q", args)
		}
	case <-time.After(time.Second):
		t.Fatal("launch request was not forwarded")
	}
}

func TestGUIInstanceForwardsFromSeparateProcess(t *testing.T) {
	if os.Getenv("DIGGERTERM_GUI_TEST_CHILD") == "1" {
		current, err := user.Current()
		if err != nil {
			t.Fatal(err)
		}
		_, primary, err := acquireGUIInstanceAt(os.Getenv("DIGGERTERM_GUI_TEST_PIPE"), current.Uid, []string{"app", "--directory", `C:\separate process`})
		if err != nil || primary {
			t.Fatalf("second process: primary=%v err=%v", primary, err)
		}
		return
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	name := `\\.\pipe\diggerterm-gui-process-test-` + time.Now().Format("150405.000000000")
	first, primary, err := acquireGUIInstanceAt(name, current.Uid, nil)
	if err != nil || !primary {
		t.Fatalf("first process: primary=%v err=%v", primary, err)
	}
	defer first.close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGUIInstanceForwardsFromSeparateProcess$")
	cmd.Env = append(os.Environ(), "DIGGERTERM_GUI_TEST_CHILD=1", "DIGGERTERM_GUI_TEST_PIPE="+name)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v: %s", err, output)
	}
	select {
	case args := <-first.launches:
		if len(args) != 3 || args[2] != `C:\separate process` {
			t.Fatalf("forwarded: %q", args)
		}
	case <-time.After(time.Second):
		t.Fatal("child launch request not received")
	}
}
