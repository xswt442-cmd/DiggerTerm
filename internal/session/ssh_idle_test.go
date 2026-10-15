//go:build windows

package session

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestWindowsSSHIdleLoopback(t *testing.T) {
	minutes, err := strconv.Atoi(os.Getenv("DIGGERTERM_SSH_IDLE_MINUTES"))
	if minutes != 1 && minutes != 60 {
		t.Skip("set DIGGERTERM_SSH_IDLE_MINUTES=1 for a short check or 60 for concurrent 15/30/60-minute checks")
	}

	_, userPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	server := startSSHIdleShell(t, userPrivate)
	address := server.listener.Addr().(*net.TCPAddr)
	identity := filepath.Join(t.TempDir(), "client_key")
	block, err := ssh.MarshalPrivateKey(userPrivate, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(identity, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	if err := secureSSHIdleIdentity(identity); err != nil {
		t.Fatalf("restrict temporary client key permissions: %v", err)
	}
	keepaliveConfig, err := writeSSHIdleConfig(t, address, server.hostKey, identity, true)
	if err != nil {
		t.Fatal(err)
	}
	noKeepaliveConfig, err := writeSSHIdleConfig(t, address, server.hostKey, identity, false)
	if err != nil {
		t.Fatal(err)
	}

	pairs := make(map[int][]*sshIdlePair)
	durations := []int{minutes}
	if minutes == 60 {
		durations = []int{15, 30, 60}
	}
	for _, duration := range durations {
		pairs[duration] = []*sshIdlePair{
			startSSHIdlePair(t, keepaliveConfig, duration, "keepalive"),
			startSSHIdlePair(t, noKeepaliveConfig, duration, "no-keepalive"),
		}
	}
	time.Sleep(2 * time.Second)
	for _, duration := range durations {
		for _, pair := range pairs[duration] {
			probeSSHIdleClients(t, pair.directIn, pair.directLines, pair.manager, pair.info.ID, 0)
		}
	}
	start := time.Now()
	for _, duration := range durations {
		deadline := start.Add(time.Duration(duration) * time.Minute)
		if wait := time.Until(deadline); wait > 0 {
			timer := time.NewTimer(wait)
			<-timer.C
		}
		for _, pair := range pairs[duration] {
			probeSSHIdleClients(t, pair.directIn, pair.directLines, pair.manager, pair.info.ID, duration)
			pair.close(t)
		}
		t.Logf("local loopback direct and managed SSH remained interactive after %d idle minutes (ServerAlive and no-ServerAlive configs)", duration)
	}
}

func secureSSHIdleIdentity(path string) error {
	user := os.Getenv("USERNAME")
	if user == "" {
		return fmt.Errorf("USERNAME is unavailable")
	}
	return exec.Command("icacls", path, "/inheritance:r", "/grant:r", user+":R").Run()
}

type sshIdlePair struct {
	direct      *exec.Cmd
	directIn    io.WriteCloser
	directLines chan string
	manager     *Manager
	info        Info
}

func startSSHIdlePair(t *testing.T, config string, minutes int, label string) *sshIdlePair {
	t.Helper()
	spec, err := sshCommand("diggerterm-idle-test", 0)
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{"-F", config}, spec.args...)
	direct := exec.Command(spec.path, args...)
	directIn, err := direct.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	directOut, err := direct.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	directErr, err := direct.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := direct.Start(); err != nil {
		t.Fatalf("start direct OpenSSH client for %d-minute check: %v", minutes, err)
	}
	directLines := make(chan string, 32)
	go scanSSHIdleOutput(directOut, directLines)
	go scanSSHIdleOutput(directErr, directLines)

	m := NewManager(nil, nil)
	managedArgs := append([]string{"-F", config}, spec.args...)
	managedSpec := commandSpec{path: spec.path, args: managedArgs}
	info, err := m.createSpec(managedSpec, "SSH idle loopback "+label, "ssh", 100, 30)
	if err != nil {
		_ = direct.Process.Kill()
		_, _ = direct.Process.Wait()
		t.Fatalf("start managed OpenSSH client for %d-minute check: %v", minutes, err)
	}
	pair := &sshIdlePair{direct: direct, directIn: directIn, directLines: directLines, manager: m, info: info}
	t.Cleanup(func() { pair.cleanup() })
	return pair
}

func (pair *sshIdlePair) cleanup() {
	if pair.direct.ProcessState == nil {
		_ = pair.direct.Process.Kill()
		_ = pair.direct.Wait()
	}
	for _, item := range pair.manager.List() {
		if item.ID == pair.info.ID && item.Status == "running" {
			_ = pair.manager.Terminate(pair.info.ID)
		}
	}
}

func (pair *sshIdlePair) close(t *testing.T) {
	t.Helper()
	if err := pair.manager.Terminate(pair.info.ID); err != nil {
		t.Fatalf("terminate managed loopback SSH session: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	managerExited := false
	for time.Now().Before(deadline) {
		for _, item := range pair.manager.List() {
			if item.ID == pair.info.ID && item.Status == "exited" {
				managerExited = true
				break
			}
		}
		if managerExited {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !managerExited {
		t.Fatalf("managed SSH client did not exit after local termination")
	}
	_ = pair.directIn.Close()
	if err := pair.direct.Wait(); err != nil {
		t.Fatalf("direct OpenSSH client exit: %v", err)
	}
}

type sshIdleServer struct {
	listener  net.Listener
	hostKey   ssh.PublicKey
	mu        sync.Mutex
	conns     map[net.Conn]struct{}
	processes []*exec.Cmd
	closed    bool
}

func startSSHIdleShell(t *testing.T, userPrivate ed25519.PrivateKey) *sshIdleServer {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	userKey, err := ssh.NewPublicKey(userPrivate.Public())
	if err != nil {
		t.Fatal(err)
	}
	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), userKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("untrusted test client key")
		},
	}
	serverConfig.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &sshIdleServer{listener: listener, hostKey: signer.PublicKey(), conns: make(map[net.Conn]struct{})}
	t.Cleanup(server.close)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.mu.Lock()
			if server.closed {
				server.mu.Unlock()
				_ = conn.Close()
				continue
			}
			server.conns[conn] = struct{}{}
			server.mu.Unlock()
			go serveSSHIdleShell(server, conn, serverConfig)
		}
	}()
	return server
}

func (server *sshIdleServer) close() {
	_ = server.listener.Close()
	server.mu.Lock()
	server.closed = true
	for conn := range server.conns {
		_ = conn.Close()
	}
	processes := append([]*exec.Cmd(nil), server.processes...)
	server.mu.Unlock()
	for _, process := range processes {
		if process.Process != nil {
			_ = process.Process.Kill()
		}
	}
}

func serveSSHIdleShell(owner *sshIdleServer, conn net.Conn, config *ssh.ServerConfig) {
	defer func() {
		_ = conn.Close()
		owner.mu.Lock()
		delete(owner.conns, conn)
		owner.mu.Unlock()
	}()
	server, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer server.Close()
	go ssh.DiscardRequests(requests)
	for incoming := range channels {
		if incoming.ChannelType() != "session" {
			_ = incoming.Reject(ssh.UnknownChannelType, "session channel required")
			continue
		}
		stream, requests, err := incoming.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer stream.Close()
			for request := range requests {
				switch request.Type {
				case "pty-req", "window-change":
					_ = request.Reply(true, nil)
				case "shell":
					cmd := exec.Command("cmd.exe", "/D")
					cmd.Stdin, cmd.Stdout, cmd.Stderr = stream, stream, stream
					if err := cmd.Start(); err != nil {
						_ = request.Reply(false, nil)
						return
					}
					owner.mu.Lock()
					owner.processes = append(owner.processes, cmd)
					owner.mu.Unlock()
					_ = request.Reply(true, nil)
					if err := cmd.Wait(); err == nil {
						_, _ = stream.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					}
					return
				default:
					_ = request.Reply(false, nil)
				}
			}
		}()
	}
}

func writeSSHIdleConfig(t *testing.T, address *net.TCPAddr, hostKey ssh.PublicKey, identity string, keepalive bool) (string, error) {
	t.Helper()
	dir := t.TempDir()
	knownHosts := filepath.Join(dir, "known_hosts")
	knownHost := knownhosts.Line([]string{fmt.Sprintf("[%s]:%d", address.IP, address.Port)}, hostKey) + "\n"
	if err := os.WriteFile(knownHosts, []byte(knownHost), 0600); err != nil {
		return "", err
	}
	pathForConfig := func(path string) string { return strings.ReplaceAll(filepath.ToSlash(path), "\"", "\\\"") }
	config := filepath.Join(dir, "config")
	contents := fmt.Sprintf("Host diggerterm-idle-test\n  HostName %s\n  Port %d\n  User test\n  BatchMode yes\n  IdentitiesOnly yes\n  IdentityFile \"%s\"\n  StrictHostKeyChecking yes\n  UserKnownHostsFile \"%s\"\n  GlobalKnownHostsFile \"%s\"\n  UpdateHostKeys no\n", address.IP, address.Port, pathForConfig(identity), pathForConfig(knownHosts), pathForConfig(knownHosts))
	if keepalive {
		contents += "  ServerAliveInterval 15\n  ServerAliveCountMax 3\n"
	}
	if err := os.WriteFile(config, []byte(contents), 0600); err != nil {
		return "", err
	}
	return config, nil
}

func scanSSHIdleOutput(reader io.Reader, lines chan<- string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024), 64*1024)
	for scanner.Scan() {
		select {
		case lines <- scanner.Text():
		default:
		}
	}
}

func probeSSHIdleClients(t *testing.T, directIn io.WriteCloser, directLines <-chan string, m *Manager, id string, milestone int) {
	t.Helper()
	marker := fmt.Sprintf("DIGGERTERM_IDLE_PROBE_%d", milestone)
	if milestone == 0 {
		const setup = "set \"DTRM_PREFIX=DIGGERTERM_IDLE_PROBE_\"\r\n"
		if _, err := io.WriteString(directIn, setup); err != nil {
			t.Fatalf("prepare direct SSH probe at minute %d: %v", milestone, err)
		}
		if err := m.Write(id, setup); err != nil {
			t.Fatalf("prepare managed SSH probe at minute %d: %v", milestone, err)
		}
	}
	if _, err := fmt.Fprintf(directIn, "echo %%DTRM_PREFIX%%%d\r\n", milestone); err != nil {
		t.Fatalf("direct SSH client input closed at minute %d: %v", milestone, err)
	}
	directDeadline := time.After(20 * time.Second)
	directFound := false
	for !directFound {
		select {
		case line, ok := <-directLines:
			if !ok {
				t.Fatalf("direct SSH output closed at minute %d", milestone)
			}
			if strings.Contains(line, marker) {
				directFound = true
			}
		case <-directDeadline:
			t.Fatalf("direct SSH did not echo the local probe at minute %d", milestone)
		}
	}

	if err := m.Write(id, fmt.Sprintf("echo %%DTRM_PREFIX%%%d\r\n", milestone)); err != nil {
		t.Fatalf("write managed SSH probe at minute %d: %v", milestone, err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		reply, err := m.Snapshot(id)
		if err != nil {
			t.Fatalf("managed SSH snapshot at minute %d: %v", milestone, err)
		}
		data, err := base64.StdEncoding.DecodeString(reply.Data)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(marker)) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("managed SSH did not echo the local probe at minute %d", milestone)
}
