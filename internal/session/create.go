package session

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	pty "github.com/aymanbagabas/go-pty"
)

func (m *Manager) Create(profile string, cols, rows int) (Info, error) {
	return m.CreateLocal(profile, "", cols, rows)
}

// CreateLocal uses the child's actual OS working directory, never a shell
// command. It is only supported for local shells, not WSL or SSH.
func (m *Manager) CreateLocal(profile, directory string, cols, rows int) (Info, error) {
	spec, ok := profileCommand(profile)
	if !ok {
		return Info{}, fmt.Errorf("unsupported profile: %q", profile)
	}
	if !validSize(cols, rows) {
		return Info{}, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	if directory != "" {
		if profile != "powershell" && profile != "cmd" && profile != "shell" && profile != "bash" {
			return Info{}, fmt.Errorf("custom working directory is not supported for %q", profile)
		}
		if !filepath.IsAbs(directory) {
			return Info{}, errors.New("working directory must be an absolute path")
		}
		stat, err := os.Stat(directory)
		if err != nil {
			return Info{}, fmt.Errorf("working directory: %w", err)
		}
		if !stat.IsDir() {
			return Info{}, errors.New("working directory is not a directory")
		}
		spec.dir = filepath.Clean(directory)
	}
	return m.createSpec(spec, profileName(profile), profile, cols, rows)
}

// CreateSSH delegates authentication, host-key verification and ~/.ssh/config
// handling to the platform OpenSSH client. It owns ONLY the client PTY.
func (m *Manager) CreateSSH(target string, port, cols, rows int) (Info, error) {
	return m.CreateSSHWithOptions(target, port, SSHOptions{}, cols, rows)
}

// CreateSSHWithOptions adds only fixed, validated OpenSSH keepalive options.
// Authentication, host-key verification and user ssh_config remain delegated
// to OpenSSH, and the default CreateSSH behavior stays unchanged.
func (m *Manager) CreateSSHWithOptions(target string, port int, options SSHOptions, cols, rows int) (Info, error) {
	if !validSize(cols, rows) {
		return Info{}, fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	spec, err := sshCommandWithOptions(target, port, options)
	if err != nil {
		return Info{}, err
	}
	info, err := m.createSpec(spec, "SSH · "+target, "ssh", cols, rows)
	if err != nil {
		return Info{}, err
	}
	managed, err := m.get(info.ID)
	if err != nil {
		return Info{}, err
	}
	managed.mu.Lock()
	managed.sshTarget, managed.sshPort = target, port
	managed.mu.Unlock()
	return info, nil
}

func (m *Manager) createSpec(spec commandSpec, name, profile string, cols, rows int) (Info, error) {
	terminal, err := pty.New()
	if err != nil {
		return Info{}, fmt.Errorf("create PTY: %w", err)
	}
	if err = terminal.Resize(cols, rows); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("resize PTY: %w", err)
	}
	if spec.dir != "" {
		// go-pty resolves a relative executable against Cmd.Dir, unlike
		// ordinary PATH lookup; make the executable absolute first.
		resolved, resolveErr := exec.LookPath(spec.path)
		if resolveErr != nil {
			_ = terminal.Close()
			return Info{}, fmt.Errorf("find shell executable: %w", resolveErr)
		}
		spec.path = resolved
	}
	cmd := terminal.Command(spec.path, spec.args...)
	cmd.Dir = spec.dir
	if len(spec.env) != 0 {
		cmd.Env = append(os.Environ(), spec.env...)
	}
	if err = cmd.Start(); err != nil {
		_ = terminal.Close()
		return Info{}, fmt.Errorf("start %s: %w", spec.path, err)
	}

	id := m.instanceID + "-" + strconv.FormatUint(m.nextID.Add(1), 10)
	info := Info{
		ID: id, Name: name, Profile: profile, Status: "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Revision: 1,
	}
	if cmd.Process != nil {
		info.PID = cmd.Process.Pid
	}
	s := &managed{info: info, pty: terminal, cmd: cmd, readDone: make(chan struct{})}
	m.mu.Lock()
	m.sessions[id] = s
	m.order = append(m.order, id)
	m.mu.Unlock()

	go m.readLoop(s)
	go m.waitLoop(s)
	return info, nil
}

func profileName(id string) string {
	for _, p := range Profiles() {
		if p.ID == id {
			return p.Name
		}
	}
	return id
}
