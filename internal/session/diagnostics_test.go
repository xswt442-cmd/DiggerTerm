package session

import (
	"encoding/base64"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestExitMetadataAndExitCallback(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(fmt.Sprintf("exit-%d", code), func(t *testing.T) {
			exited := make(chan Info, 2)
			manager := NewManager(nil, func(info Info) { exited <- info })
			states := make(chan Info, 2)
			manager.SetStateHandler(func(info Info) { states <- info })
			live, err := manager.Create(Profiles()[0].ID, 80, 24)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if current, ok := findInfo(manager.List(), live.ID); ok && current.Status == "running" {
					_ = manager.Terminate(live.ID)
					waitForExitInfo(t, exited, live.ID)
				}
			})

			started, err := manager.createSpec(exitCommand(code), "exit diagnostics", "shell", 80, 24)
			if err != nil {
				t.Fatal(err)
			}
			if started.Revision != 1 {
				t.Fatalf("initial revision = %d, want 1", started.Revision)
			}
			if _, err := time.Parse(time.RFC3339Nano, started.StartedAt); err != nil {
				t.Fatalf("startedAt is not RFC3339Nano: %q: %v", started.StartedAt, err)
			}
			final := waitForExitInfo(t, exited, started.ID)
			assertExitMetadata(t, started, final, code, "process-exit")
			select {
			case stream := <-states:
				if stream.ID != started.ID || !stream.StreamClosed || stream.Revision >= final.Revision {
					t.Fatalf("stream closure event did not precede process exit metadata: %+v", stream)
				}
			case <-time.After(time.Second):
				t.Fatal("missing stream closure event")
			}
			listed, ok := findInfo(manager.List(), started.ID)
			if !ok || listed.Revision != final.Revision || listed.ExitCode == nil || *listed.ExitCode != code {
				t.Fatalf("List did not preserve exit metadata: %+v", listed)
			}
			if current, ok := findInfo(manager.List(), live.ID); !ok || current.Status != "running" {
				t.Fatalf("one process exit affected another session: %+v", current)
			}
		})
	}
}

func TestRenameAndUserTerminationAdvanceRevision(t *testing.T) {
	exited := make(chan Info, 1)
	manager := NewManager(nil, func(info Info) { exited <- info })
	info, err := manager.Create(Profiles()[0].ID, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if current, ok := findInfo(manager.List(), info.ID); ok && current.Status == "running" {
			_ = manager.Terminate(info.ID)
			waitForExitInfo(t, exited, info.ID)
		}
	})
	rename, err := manager.Rename(info.ID, "diagnostics")
	if err != nil {
		t.Fatal(err)
	}
	if rename.Revision != info.Revision+1 {
		t.Fatalf("rename revision %d did not increment from %d", rename.Revision, info.Revision)
	}
	if err := manager.Terminate(info.ID); err != nil {
		t.Fatal(err)
	}
	final := waitForExitInfo(t, exited, info.ID)
	if final.ExitReason != "user-terminated" || final.Status != "exited" || final.Revision <= rename.Revision {
		t.Fatalf("termination metadata did not advance: %+v", final)
	}
	if final.ExitCode == nil {
		t.Fatal("ProcessState exit code was not captured")
	}
}

func TestSnapshotReportsBoundedReplayTruncation(t *testing.T) {
	exited := make(chan Info, 1)
	manager := NewManager(nil, func(info Info) { exited <- info })
	info, err := manager.createSpec(outputCommand(), "bounded replay", "shell", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	waitForExitInfo(t, exited, info.ID)
	replay, err := manager.Snapshot(info.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := base64.StdEncoding.DecodeString(replay.Data)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Truncated || replay.Offset <= uint64(len(data)) || len(data) != maxReplayBytes {
		t.Fatalf("bounded snapshot metadata is inaccurate: offset=%d bytes=%d truncated=%v", replay.Offset, len(data), replay.Truncated)
	}
}

func assertExitMetadata(t *testing.T, started, final Info, code int, reason string) {
	t.Helper()
	if final.Status != "exited" || final.ExitCode == nil || *final.ExitCode != code || final.ExitReason != reason {
		t.Fatalf("wrong process exit metadata: %+v", final)
	}
	if !final.StreamClosed || final.Revision <= started.Revision || final.ExitedAt == "" {
		t.Fatalf("stream/status metadata did not advance: %+v", final)
	}
	if final.StreamError != "" {
		t.Fatalf("normal process exit reported a PTY stream failure: %q", final.StreamError)
	}
	if _, err := time.Parse(time.RFC3339Nano, final.ExitedAt); err != nil {
		t.Fatalf("exitedAt is not RFC3339Nano: %q: %v", final.ExitedAt, err)
	}
}

func waitForExitInfo(t *testing.T, exited <-chan Info, id string) Info {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for {
		select {
		case info := <-exited:
			if info.ID == id {
				return info
			}
		case <-deadline:
			t.Fatalf("session %s did not emit exit metadata", id)
			return Info{}
		}
	}
}

func findInfo(infos []Info, id string) (Info, bool) {
	for _, info := range infos {
		if info.ID == id {
			return info, true
		}
	}
	return Info{}, false
}

func exitCommand(code int) commandSpec {
	if runtime.GOOS == "windows" {
		return commandSpec{path: "cmd.exe", args: []string{"/D", "/C", fmt.Sprintf("exit /b %d", code)}}
	}
	return commandSpec{path: "/bin/sh", args: []string{"-c", fmt.Sprintf("exit %d", code)}}
}

func outputCommand() commandSpec {
	if runtime.GOOS == "windows" {
		line := "01234567890123456789012345678901234567890123456789012345678901234567890123456789"
		return commandSpec{path: "cmd.exe", args: []string{"/D", "/C", fmt.Sprintf("for /L %%i in (1,1,5000) do @echo %s", line)}}
	}
	return commandSpec{path: "/bin/sh", args: []string{"-c", "i=0; while [ $i -lt 5000 ]; do printf '%080d\\n' 1; i=$((i+1)); done"}}
}
