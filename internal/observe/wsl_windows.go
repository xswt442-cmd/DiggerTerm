//go:build windows

package observe

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

const maxWSLDistros = 4

// Fixed, parameter-free POSIX shell fragment: only /proc, id and ps metadata.
// It neither reads environ/cmdline nor accesses terminal I/O or signals.
// The default distro user is used: there is no -u root or elevation.
const linuxProcessScript = `uid=$(id -u) || exit 2
hz=$(getconf CLK_TCK) || exit 2
btime=
while read -r field value rest; do
  if [ "$field" = btime ]; then btime=$value; break; fi
done < /proc/stat
IFS= read -r bootid < /proc/sys/kernel/random/boot_id || exit 2
[ -n "$btime" ] && [ -n "$hz" ] && [ -n "$bootid" ] || exit 2
printf 'H\t%s\t%s\t%s\t%s\n' "$btime" "$hz" "$bootid" "$$"
ps -eo pid=,ppid=,uid=,comm= |
while read -r pid ppid owner comm; do
  [ "$owner" = "$uid" ] || continue
  case "$comm" in ''|*[!a-zA-Z0-9_.+-]*) continue;; esac
  [ -r "/proc/$pid/stat" ] || continue
  IFS= read -r data < "/proc/$pid/stat" 2>/dev/null || continue
  data=${data##*) }
  set -- $data
  [ $# -ge 20 ] || continue
  printf 'P\t%s\t%s\t%s\t%s\n' "$pid" "$ppid" "${20}" "$comm"
done`

func wslExecutable() (string, error) {
	root := os.Getenv("SystemRoot")
	if root == "" {
		return "", errors.New("Windows SystemRoot unavailable")
	}
	exe := filepath.Join(root, "System32", "wsl.exe")
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("WSL executable unavailable: %w", err)
	}
	return exe, nil
}

func parseRunningDistros(data []byte) []string {
	// wsl.exe --list --running --quiet emits UTF-16LE on Windows.
	if len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe {
		data = data[2:]
	}
	if bytes.IndexByte(data, 0) >= 0 {
		if len(data)%2 != 0 {
			data = data[:len(data)-1]
		}
		u := make([]uint16, len(data)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(data[2*i:])
		}
		data = []byte(string(utf16.Decode(u)))
	}
	names := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range strings.Split(string(data), "\n") {
		name := strings.TrimSpace(strings.TrimPrefix(item, "\ufeff"))
		if name == "" || strings.HasPrefix(strings.ToLower(name), "docker-desktop") || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func runWSLCommand(ctx context.Context, exe string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	var out cappedBuffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	if out.exceeded {
		return nil, errors.New("WSL response exceeded size limit")
	}
	return out.Bytes(), nil
}

// Each WSL invocation is short and bounded. No stopped distro is launched.
func platformWSLSnapshot() ([]Process, bool, string) {
	exe, err := wslExecutable()
	if err != nil {
		return nil, false, "WSL discovery unavailable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	data, err := runWSLCommand(ctx, exe, "--list", "--running", "--quiet")
	cancel()
	if err != nil {
		return nil, false, "Unable to list running WSL distributions"
	}
	distros := parseRunningDistros(data)
	if len(distros) == 0 {
		return nil, false, "No running WSL user distributions"
	}
	truncated := false
	if len(distros) > maxWSLDistros {
		distros = distros[:maxWSLDistros]
		truncated = true
	}
	observed := make([]Process, 0)
	failures := 0
	for _, distro := range distros {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		raw, err := runWSLCommand(ctx, exe, "-d", distro, "--exec", "/bin/sh", "-c", linuxProcessScript)
		cancel()
		if err != nil {
			failures++
			continue
		}
		processes, clipped, err := parseWSLProcessList(raw, distro)
		if err != nil {
			failures++
			continue
		}
		observed = append(observed, processes...)
		truncated = truncated || clipped
	}
	note := fmt.Sprintf("%d running WSL distro(s) scanned", len(distros)-failures)
	if failures > 0 {
		note += fmt.Sprintf("; %d unavailable", failures)
	}
	return observed, truncated, note
}

// cappedBuffer returns an error via a flag, not from Write: aborting a pipe
// early can leave wsl.exe waiting for its Linux child to flush stdout.
type cappedBuffer struct {
	bytes.Buffer
	exceeded bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	const maxBytes = 256 * 1024
	if c.Len()+len(p) > maxBytes {
		c.exceeded = true
		available := maxBytes - c.Len()
		if available > 0 {
			_, _ = c.Buffer.Write(p[:available])
		}
		return len(p), nil
	}
	return c.Buffer.Write(p)
}
