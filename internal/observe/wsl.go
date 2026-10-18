package observe

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WSL process identifiers are separate from Windows PIDs. The WSL boot UUID
// and kernel start ticks disambiguate PID reuse, restarts, and distributions.
func parseWSLProcessList(data []byte, distro string) ([]Process, bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 16*1024)
	if !scanner.Scan() {
		return nil, false, fmt.Errorf("WSL %s: missing process snapshot header", distro)
	}
	header := strings.Split(scanner.Text(), "\t")
	if len(header) != 5 || header[0] != "H" {
		return nil, false, fmt.Errorf("WSL %s: invalid process header", distro)
	}
	bootSeconds, err1 := strconv.ParseInt(header[1], 10, 64)
	hz, err2 := strconv.ParseInt(header[2], 10, 64)
	scannerPID, err3 := strconv.Atoi(header[4])
	bootID := strings.ToLower(header[3])
	if err1 != nil || err2 != nil || err3 != nil || bootSeconds < 1 || hz < 1 ||
		hz > 1_000_000 || !validBootID(bootID) || scannerPID < 1 {
		return nil, false, fmt.Errorf("WSL %s: invalid Linux boot metadata", distro)
	}

	rows := make([]processRow, 0, 64)
	ticksByPID := make(map[int]uint64)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "\t")
		if len(line) != 5 || line[0] != "P" {
			continue
		}
		pid, e1 := strconv.Atoi(line[1])
		ppid, e2 := strconv.Atoi(line[2])
		ticks, e3 := strconv.ParseUint(line[3], 10, 64)
		name := line[4]
		if e1 != nil || e2 != nil || e3 != nil || pid < 1 || ppid < 0 || ticks == 0 ||
			len(name) == 0 || len(name) > 64 || !validLinuxName(name) {
			continue
		}
		if len(rows) >= 2048 {
			return nil, false, fmt.Errorf("WSL %s: too many process records", distro)
		}
		sec := bootSeconds + int64(ticks/uint64(hz))
		nsec := int64(ticks%uint64(hz)) * int64(time.Second) / hz
		rows = append(rows, processRow{
			pid: pid, parentPID: ppid, name: name,
			started: time.Unix(sec, nsec).UTC(),
		})
		ticksByPID[pid] = ticks
	}
	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("WSL %s: read snapshot: %w", distro, err)
	}
	// Scanner sh and its transient ps/child processes are not external
	// observations; remove that process tree from our own snapshot.
	result, truncated := selectRelatedWithKind(rows, []int{scannerPID}, maxObserved, linuxRootKind)
	for i := range result {
		p := &result[i]
		p.Source = "wsl"
		p.Distribution = distro
		p.Key = fmt.Sprintf("wsl:%s:%s:%d:%d", distro, bootID, p.PID, ticksByPID[p.PID])
	}
	return result, truncated, nil
}

func wslNamespace(data []byte, distro string) string {
	line, _, _ := bytes.Cut(data, []byte("\n"))
	fields := strings.Split(strings.TrimSuffix(string(line), "\r"), "\t")
	if len(fields) != 5 || fields[0] != "H" || !validBootID(strings.ToLower(fields[3])) {
		return ""
	}
	return "wsl:" + distro + ":" + strings.ToLower(fields[3])
}

func validBootID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func validLinuxName(s string) bool {
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == '+') {
			return false
		}
	}
	return true
}

func linuxRootKind(name string) string {
	switch strings.ToLower(name) {
	case "bash", "zsh", "fish", "sh", "dash", "ksh", "tmux", "screen", "pwsh", "ssh", "sshd", "nu":
		return "terminal"
	case "claude", "codex", "gemini", "opencode", "aider":
		return "agent"
	}
	return ""
}
