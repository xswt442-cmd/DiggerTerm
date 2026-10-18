// Package observe exposes a strictly read-only process inventory.
// Unlike managed sessions, observed processes have NO control API.
package observe

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxObserved = 160

type Process struct {
	Key                 string `json:"key"` // Namespace + PID + start time; not an authorization token
	PID                 int    `json:"pid"`
	ParentPID           int    `json:"parentPid"`
	Name                string `json:"name"`
	Kind                string `json:"kind"`                          // terminal, agent, or descendant
	ClassificationBasis string `json:"classificationBasis,omitempty"` // name or ancestor
	Source              string `json:"source"`                        // windows or wsl
	Distribution        string `json:"distribution"`                  // only populated for WSL
	StartedAt           string `json:"startedAt"`                     // UTC RFC3339, known process start time
}

type Snapshot struct {
	Supported bool           `json:"supported"`
	ScannedAt string         `json:"scannedAt"`
	Processes []Process      `json:"processes"`
	Sources   []SourceStatus `json:"sources"`
	Truncated bool           `json:"truncated"`
	Note      string         `json:"note"`
}

type SourceStatus struct {
	Source       string `json:"source"`
	Distribution string `json:"distribution"`
	Namespace    string `json:"namespace"`
	ScannedAt    string `json:"scannedAt"`
	State        string `json:"state"` // success, partial, failed, unsupported, not-running, or skipped
	Complete     bool   `json:"complete"`
	Truncated    bool   `json:"truncated"`
	Policy       string `json:"policy"`
	Reason       string `json:"reason"`
}

// Scan never opens PTY handles and never reads process command lines,
// environments, terminal contents, or other private process memory.
func Scan(ownedPIDs []int) (Snapshot, error) {
	return ScanContext(context.Background(), ownedPIDs)
}

// ScanContext inventories process metadata and stops when its caller cancels.
// Platform subprocesses, including WSL scans, inherit this context.
func ScanContext(ctx context.Context, ownedPIDs []int) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	rows, supported, err := platformSnapshot(ctx)
	if err != nil && ctx.Err() != nil {
		return Snapshot{}, ctx.Err()
	}
	if err != nil {
		rows = nil
	}
	scannedAt := time.Now().UTC().Format(time.RFC3339Nano)
	result := Snapshot{
		Supported: supported,
		ScannedAt: scannedAt,
		Processes: make([]Process, 0),
		Sources:   make([]SourceStatus, 0),
		Note:      "Process-name heuristics only. No screen contents or control permissions.",
	}
	if !supported {
		result.Note = "Read-only external process scanning is available on Windows only."
		result.Sources = append(result.Sources, SourceStatus{
			Source: "windows", ScannedAt: scannedAt, State: "unsupported", Policy: "windows-same-user-metadata-v1", Reason: "platform-unsupported",
		})
		return result, nil
	}
	win, windowsClipped := selectRelated(rows, ownedPIDs, maxObserved)
	for i := range win {
		win[i].Source = "windows"
	}
	windowsStatus := SourceStatus{
		Source: "windows", Namespace: platformNamespace(), ScannedAt: scannedAt,
		State: "success", Complete: !windowsClipped, Truncated: windowsClipped,
		Policy: "windows-same-user-query-limited-v1",
	}
	if err != nil {
		windowsStatus.State = "failed"
		windowsStatus.Complete = false
		windowsStatus.Reason = "process-enumeration-failed"
	}
	if windowsStatus.Namespace == "" && err == nil {
		windowsStatus.Complete = false
		windowsStatus.Reason = "namespace-unavailable"
	}
	result.Sources = append(result.Sources, windowsStatus)
	wsl, wslClipped, wslNote, wslStatuses := platformWSLResultsContext(ctx)
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	for i := range wslStatuses {
		wslStatuses[i].ScannedAt = scannedAt
	}
	result.Sources = append(result.Sources, wslStatuses...)
	// The two OS namespaces can legitimately have the same numeric PID.
	// Keep Windows and WSL processes in one explicitly scoped read-only list.
	result.Processes = mergeObserved(win, wsl, maxObserved)
	result.Truncated = windowsClipped || wslClipped || len(win)+len(wsl) > maxObserved
	if len(win)+len(wsl) > maxObserved {
		for i := range result.Sources {
			if result.Sources[i].State == "success" {
				result.Sources[i].Complete = false
				result.Sources[i].Truncated = true
				result.Sources[i].Reason = "combined-result-limit"
			}
		}
	}
	if wslNote != "" {
		result.Note += " " + wslNote + "."
	}
	return result, nil
}

type processRow struct {
	pid       int
	parentPID int
	name      string
	started   time.Time
}

// Deliberately conservative heuristics: an agent using a generic node.exe
// process can only be shown as a descendant of a recognized process.
func rootKind(name string) string {
	switch strings.ToLower(name) {
	case "cmd.exe", "powershell.exe", "pwsh.exe", "wsl.exe", "wslhost.exe",
		"bash.exe", "ssh.exe", "wt.exe", "windowsterminal.exe", "openconsole.exe",
		"conhost.exe", "mintty.exe", "wezterm-gui.exe", "alacritty.exe",
		"hyper.exe", "tabby.exe":
		return "terminal"
	case "claude.exe", "codex.exe", "gemini.exe", "opencode.exe", "aider.exe":
		return "agent"
	}
	return ""
}

func selectRelated(rows []processRow, managed []int, limit int) ([]Process, bool) {
	return selectRelatedWithKind(rows, managed, limit, rootKind)
}

func selectRelatedWithKind(rows []processRow, managed []int, limit int, classify func(string) string) ([]Process, bool) {
	byPID := make(map[int]processRow, len(rows))
	for _, row := range rows {
		if row.pid > 0 && !row.started.IsZero() {
			byPID[row.pid] = row
		}
	}
	owned := make(map[int]bool, len(managed))
	for _, pid := range managed {
		owned[pid] = true
	}

	// Every parent link must be newer than neither child nor its own start:
	// this prevents attaching a child to an unrelated PID reused by Windows.
	parentOf := func(row processRow) (processRow, bool) {
		parent, ok := byPID[row.parentPID]
		return parent, ok && parent.pid != row.pid && !parent.started.After(row.started)
	}
	// Determine whether a process belongs to our PTY tree, even when its
	// descendant executable looks like an external agent.
	isOwned := func(row processRow) bool {
		seen := make(map[int]bool)
		for {
			if owned[row.pid] {
				return true
			}
			if seen[row.pid] {
				return false
			}
			seen[row.pid] = true
			parent, ok := parentOf(row)
			if !ok {
				return false
			}
			row = parent
		}
	}
	relatedKind := func(row processRow) string {
		if kind := classify(row.name); kind != "" {
			return kind
		}
		seen := make(map[int]bool)
		for {
			if seen[row.pid] {
				break
			}
			seen[row.pid] = true
			parent, ok := parentOf(row)
			if !ok {
				break
			}
			if classify(parent.name) != "" {
				return "descendant"
			}
			row = parent
		}
		return ""
	}

	processes := make([]Process, 0)
	for _, row := range rows {
		if row.pid <= 0 || row.started.IsZero() || isOwned(row) {
			continue
		}
		kind := relatedKind(row)
		if kind == "" {
			continue
		}
		basis := "name"
		if kind == "descendant" {
			basis = "ancestor"
		}
		processes = append(processes, Process{
			Key:                 fmt.Sprintf("windows:%d:%d", row.pid, row.started.UnixNano()),
			PID:                 row.pid,
			ParentPID:           row.parentPID,
			Name:                row.name,
			Kind:                kind,
			ClassificationBasis: basis,
			StartedAt:           row.started.UTC().Format(time.RFC3339Nano),
		})
	}
	// Prioritize explicitly recognized terminals/agents before descendants;
	// a large process tree should not hide the terminal roots.
	sort.Slice(processes, func(i, j int) bool {
		a, b := processes[i], processes[j]
		if a.Kind != b.Kind {
			if a.Kind == "descendant" {
				return false
			}
			if b.Kind == "descendant" {
				return true
			}
		}
		if a.Name != b.Name {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return a.PID < b.PID
	})
	if limit < 0 {
		limit = 0
	}
	truncated := len(processes) > limit
	if truncated {
		processes = processes[:limit]
	}
	return processes, truncated
}
