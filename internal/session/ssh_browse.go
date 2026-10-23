package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxBrowseOutput  = 192 * 1024
	maxBrowseEntries = 800
	sshBrowseTimeout = 10 * time.Second
)

type SSHDirectoryListing struct {
	Path        string   `json:"path"`
	Directories []string `json:"directories"`
	Truncated   bool     `json:"truncated"`
}

func (m *Manager) BrowseSSHDirectories(ctx context.Context, id, directory string) (SSHDirectoryListing, error) {
	s, err := m.get(id)
	if err != nil {
		return SSHDirectoryListing{}, err
	}
	s.mu.Lock()
	info, target, port := s.info, s.sshTarget, s.sshPort
	s.mu.Unlock()
	if info.Profile != "ssh" || target == "" {
		return SSHDirectoryListing{}, errors.New("directory browsing is supported only for managed SSH sessions")
	}
	if info.Status != "running" {
		return SSHDirectoryListing{}, ErrNotRunning
	}
	if err := validateSSHTarget(target); err != nil {
		return SSHDirectoryListing{}, err
	}
	command, err := sshBrowseCommand(directory)
	if err != nil {
		return SSHDirectoryListing{}, err
	}
	browseCtx, cancel := context.WithTimeout(ctx, sshBrowseTimeout)
	defer cancel()
	program := "ssh"
	if runtime.GOOS == "windows" {
		program = "ssh.exe"
	}
	path, err := exec.LookPath(program)
	if err != nil {
		return SSHDirectoryListing{}, fmt.Errorf("OpenSSH not found: %w", err)
	}
	args := []string{"-T", "-o", "BatchMode=yes", "-o", "NumberOfPasswordPrompts=0", "-o", "ConnectTimeout=5", "-o", "ClearAllForwardings=yes", "-o", "ForwardAgent=no", "-o", "ForwardX11=no", "-o", "PermitLocalCommand=no"}
	if port != 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, target, command)
	cmd := exec.CommandContext(browseCtx, path, args...)
	stdout, stderr := &boundedBrowseOutput{}, &boundedBrowseOutput{limit: 4096}
	stdout.limit = maxBrowseOutput
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	if browseCtx.Err() != nil {
		return SSHDirectoryListing{}, fmt.Errorf("remote directory listing: %w", browseCtx.Err())
	}
	if stdout.exceeded {
		return SSHDirectoryListing{}, errors.New("remote listing exceeded output limit")
	}
	if err != nil {
		// Never expose credentials, full command arguments or raw remote output.
		message := strings.TrimSpace(stderr.buf.String())
		if len(message) > 300 {
			message = message[:300]
		}
		if message == "" {
			message = "SSH failed; check key/agent authentication and remote POSIX shell support"
		}
		return SSHDirectoryListing{}, fmt.Errorf("remote directory listing failed: %s", message)
	}
	return parseSSHDirectoryListing(stdout.buf.Bytes())
}

type boundedBrowseOutput struct {
	buf      bytes.Buffer
	limit    int
	exceeded bool
}

func (w *boundedBrowseOutput) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.buf.Len() {
		w.exceeded = true
		return 0, errors.New("SSH listing output limit exceeded")
	}
	return w.buf.Write(data)
}

func sshBrowseCommand(path string) (string, error) {
	if path == "" {
		path = "."
	}
	if len(path) > 2048 || !utf8.ValidString(path) {
		return "", errors.New("invalid remote directory")
	}
	for _, char := range path {
		if char < 0x20 || char == 0x7f || (char >= 0x80 && char <= 0x9f) ||
			(char >= 0x202a && char <= 0x202e) || (char >= 0x2066 && char <= 0x2069) {
			return "", errors.New("remote directory contains control characters")
		}
	}
	if path != "." && !strings.HasPrefix(path, "/") {
		return "", errors.New("remote directory must be absolute")
	}
	// The remote default shell must be POSIX compatible. All user data is
	// passed as a single-quoted shell literal; no terminal PTY is touched.
	quoted := "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	return "LC_ALL=C; cd -- " + quoted + " || exit 42; printf '%s\\0' \"$(pwd -P)\"; for p in ./* ./.[!.]* ./..?*; do if [ -d \"$p\" ]; then printf '%s\\0' \"${p#./}\"; fi; done", nil
}

func containsRemoteControls(value string) bool {
	for _, ch := range value {
		if ch < 0x20 || ch == 0x7f || (ch >= 0x80 && ch <= 0x9f) ||
			(ch >= 0x202a && ch <= 0x202e) || (ch >= 0x2066 && ch <= 0x2069) {
			return true
		}
	}
	return false
}

func parseSSHDirectoryListing(data []byte) (SSHDirectoryListing, error) {
	if len(data) == 0 || data[len(data)-1] != 0 {
		return SSHDirectoryListing{}, errors.New("incomplete remote directory listing")
	}
	parts := bytes.Split(data[:len(data)-1], []byte{0})
	if len(parts) == 0 || len(parts[0]) > 4096 || !utf8.Valid(parts[0]) || !strings.HasPrefix(string(parts[0]), "/") || containsRemoteControls(string(parts[0])) {
		return SSHDirectoryListing{}, errors.New("remote directory listing has no absolute path")
	}
	listing := SSHDirectoryListing{Path: string(parts[0]), Directories: []string{}}
	for _, part := range parts[1:] {
		if len(part) == 0 || len(part) > 2048 || !utf8.Valid(part) {
			continue
		}
		name := string(part)
		if name == "." || name == ".." || strings.ContainsAny(name, "/\\") || containsRemoteControls(name) {
			continue
		}
		if len(listing.Directories) >= maxBrowseEntries {
			listing.Truncated = true
			break
		}
		listing.Directories = append(listing.Directories, name)
	}
	sort.Strings(listing.Directories)
	return listing, nil
}
