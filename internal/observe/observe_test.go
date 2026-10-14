package observe

import (
	"testing"
	"time"
)

func TestRelatedProcessesAndOwnership(t *testing.T) {
	t0 := time.Unix(1700000000, 0)
	rows := []processRow{
		{pid: 10, name: "pwsh.exe", started: t0},
		{pid: 11, parentPID: 10, name: "node.exe", started: t0.Add(time.Second)},
		{pid: 12, parentPID: 11, name: "python.exe", started: t0.Add(2 * time.Second)},
		{pid: 20, name: "cmd.exe", started: t0},
		{pid: 21, parentPID: 20, name: "claude.exe", started: t0.Add(time.Second)},
		{pid: 30, name: "notepad.exe", started: t0},
		{pid: 31, parentPID: 30, name: "helper.exe", started: t0.Add(time.Second)},
		// PID reuse: an old child cannot inherit newer parent's classification.
		{pid: 40, name: "pwsh.exe", started: t0.Add(20 * time.Second)},
		{pid: 41, parentPID: 40, name: "node.exe", started: t0.Add(10 * time.Second)},
	}
	got, truncated := selectRelated(rows, []int{20}, 160)
	if truncated {
		t.Fatal("unexpected truncation")
	}
	seen := map[int]string{}
	for _, item := range got {
		seen[item.PID] = item.Kind
		if item.Key == "" || item.StartedAt == "" {
			t.Fatalf("missing stable metadata: %+v", item)
		}
	}
	for pid, want := range map[int]string{10: "terminal", 11: "descendant", 12: "descendant", 40: "terminal"} {
		if seen[pid] != want {
			t.Errorf("pid=%d kind=%q, want %q", pid, seen[pid], want)
		}
	}
	for _, pid := range []int{20, 21, 30, 31, 41} {
		if _, exists := seen[pid]; exists {
			t.Errorf("unexpected observed PID %d", pid)
		}
	}
}

func TestObservedLimit(t *testing.T) {
	now := time.Unix(1700000000, 0)
	rows := []processRow{
		{pid: 8, name: "cmd.exe", started: now},
		{pid: 9, name: "pwsh.exe", started: now},
		{pid: 10, name: "node.exe", parentPID: 8, started: now.Add(time.Second)},
	}
	got, clipped := selectRelated(rows, nil, 2)
	if !clipped || len(got) != 2 {
		t.Fatalf("unexpected bound: %d clipped=%v", len(got), clipped)
	}
	for _, item := range got {
		if item.Kind == "descendant" {
			t.Fatal("descendant took a root slot")
		}
	}
}

func TestClassificationConservative(t *testing.T) {
	if rootKind("node.exe") != "" || rootKind("python.exe") != "" {
		t.Fatal("generic runtimes should not be claimed as agents")
	}
	if rootKind("CODEX.EXE") != "agent" || rootKind("pwsh.exe") != "terminal" {
		t.Fatal("expected known executable names")
	}
}
