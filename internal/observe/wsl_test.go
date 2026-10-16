package observe

import (
	"fmt"
	"strings"
	"testing"
)

const testBootID = "0e07a822-90af-4e44-a555-aaa77dfe706f"

func sampleWSLProcessLines() []byte {
	rows := []string{
		fmt.Sprintf("H\t1791967036\t100\t%s\t40", testBootID),
		"P\t40\t1\t100\tsh",      // scanner, must be excluded
		"P\t41\t40\t110\tps",     // transient child, excluded
		"P\t50\t1\t500\tbash",    // unrelated user-owned shell
		"P\t51\t50\t510\tnode",   // descendant, never assume node==AI
		"P\t52\t51\t530\tpython", // descendant
		"P\t53\t1\t600\tcodex",   // agent root
		"P\t54\t53\t620\tpython", // descendant of agent
		"P\t61\t62\t180\tnode",   // parent starts after child, stale PID reuse
		"P\t62\t1\t200\tbash",
		"P\t70\t1\t700\thtop",     // unrelated, not shown
		"P\t80\t1\t720\tbad name", // reject whitespace-control spoofing
	}
	return []byte(strings.Join(rows, "\n") + "\n")
}

func TestWSLProcessSnapshotIdentityAndOwnership(t *testing.T) {
	a, truncated, err := parseWSLProcessList(sampleWSLProcessLines(), "Ubuntu")
	if err != nil || truncated {
		t.Fatalf("parse: %v truncated=%v", err, truncated)
	}
	kinds := make(map[int]string)
	keys := make(map[int]string)
	for _, p := range a {
		kinds[p.PID] = p.Kind
		keys[p.PID] = p.Key
		if p.Source != "wsl" || p.Distribution != "Ubuntu" || p.StartedAt == "" {
			t.Fatalf("WSL scoping missing: %+v", p)
		}
	}
	for pid, want := range map[int]string{50: "terminal", 51: "descendant", 52: "descendant", 53: "agent", 54: "descendant", 62: "terminal"} {
		if kinds[pid] != want {
			t.Errorf("PID=%d got %q want %q", pid, kinds[pid], want)
		}
	}
	for _, pid := range []int{40, 41, 61, 70, 80} {
		if _, ok := kinds[pid]; ok {
			t.Errorf("unexpected process %d", pid)
		}
	}
	b, _, err := parseWSLProcessList(sampleWSLProcessLines(), "Debian")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range b {
		if p.PID == 50 && keys[p.PID] == p.Key {
			t.Fatal("WSL distro collision in stable key")
		}
	}
	newBoot := strings.Replace(string(sampleWSLProcessLines()), testBootID, "11111111-2222-4333-8444-555555555555", 1)
	c, _, err := parseWSLProcessList([]byte(newBoot), "Ubuntu")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range c {
		if p.PID == 50 && keys[p.PID] == p.Key {
			t.Fatal("WSL restart retained the prior boot identity")
		}
	}
}

func TestWSLRejectBadBootMetadata(t *testing.T) {
	for _, v := range []string{"", "H\t0\t100\t" + testBootID + "\t20", "H\t1791967036\t0\t" + testBootID + "\t20", "H\t1791967036\t100\tinvalid\t20"} {
		if _, _, err := parseWSLProcessList([]byte(v), "Ubuntu"); err == nil {
			t.Errorf("bad header accepted: %q", v)
		}
	}
}
