//go:build windows

package session

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// A real loopback SSH server checks OpenSSH <-> ConPTY interactive I/O.
// Its ephemeral host key is trusted ONLY by this test's private file;
// production CreateSSH never changes host-key verification policy.
func testSSHServer(t *testing.T) (string, int, ssh.PublicKey) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				server, channels, requests, err := ssh.NewServerConn(conn, cfg)
				if err != nil {
					return
				}
				defer server.Close()
				go ssh.DiscardRequests(requests)
				for channel := range channels {
					if channel.ChannelType() != "session" {
						_ = channel.Reject(ssh.UnknownChannelType, "session channel required")
						continue
					}
					stream, sessionRequests, err := channel.Accept()
					if err != nil {
						continue
					}
					go func() {
						defer stream.Close()
						go func() {
							for request := range sessionRequests {
								switch request.Type {
								case "pty-req", "window-change", "shell":
									_ = request.Reply(true, nil)
									if request.Type == "shell" {
										_, _ = stream.Write([]byte("DIGGERTERM_SSH_WELCOME\r\n"))
									}
								default:
									_ = request.Reply(false, nil)
								}
							}
						}()
						lines := bufio.NewReader(stream)
						for {
							line, err := lines.ReadString('\n')
							if err != nil {
								return
							}
							if strings.Contains(line, "DIGGERTERM_SSH_INPUT") {
								_, _ = stream.Write([]byte("DIGGERTERM_SSH_ECHO\r\n"))
							}
						}
					}()
				}
			}()
		}
	}()
	addr := listener.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, signer.PublicKey()
}

func waitSSHOutput(t *testing.T, m *Manager, id, expected string) {
	t.Helper()
	deadline := time.Now().Add(18 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		reply, err := m.Snapshot(id)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := base64.StdEncoding.DecodeString(reply.Data)
		if err != nil {
			t.Fatal(err)
		}
		last = string(raw)
		if strings.Contains(last, expected) {
			return
		}
		time.Sleep(55 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q; PTY output: %.2200q", expected, last)
}

func TestWindowsSSHInteractiveConPTY(t *testing.T) {
	host, port, key := testSSHServer(t)
	// Strict verification must stay enabled: only an explicitly known key
	// stored in this temp file permits connection to the local test server.
	knownFile := filepath.Join(t.TempDir(), "ssh_known_hosts")
	if err := os.WriteFile(knownFile, []byte(knownhosts.Line(
		[]string{fmt.Sprintf("[%s]:%d", host, port)}, key)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec, err := sshCommand(host, port)
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{
		"-F", "none",
		"-o", "UserKnownHostsFile=" + knownFile,
		"-o", "GlobalKnownHostsFile=" + knownFile,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "BatchMode=yes",
		"-o", "UpdateHostKeys=no",
	}, spec.args...)
	spec.args = args
	m := NewManager(nil, nil)
	info, err := m.createSpec(spec, "SSH loopback", "ssh", 95, 29)
	if err != nil {
		t.Fatalf("starting OpenSSH in ConPTY: %v", err)
	}
	defer func() {
		for _, item := range m.List() {
			if item.ID == info.ID && item.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()
	if info.PID < 1 || info.Profile != "ssh" {
		t.Fatalf("invalid SSH session metadata: %+v", info)
	}
	waitSSHOutput(t, m, info.ID, "DIGGERTERM_SSH_WELCOME")
	if err := m.Resize(info.ID, 120, 37); err != nil {
		t.Fatalf("resize remote terminal: %v", err)
	}
	if err := m.Write(info.ID, "DIGGERTERM_SSH_INPUT\r\n"); err != nil {
		t.Fatalf("write SSH: %v", err)
	}
	waitSSHOutput(t, m, info.ID, "DIGGERTERM_SSH_ECHO")
	if err := m.Terminate(info.ID); err != nil {
		t.Fatalf("explicit terminate SSH: %v", err)
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if m.List()[0].Status == "exited" {
			if err := m.Forget(info.ID); err != nil {
				t.Fatalf("forget SSH session: %v", err)
			}
			return
		}
		time.Sleep(70 * time.Millisecond)
	}
	t.Fatal("SSH client did not exit after explicit termination")
}

func TestWindowsSSHFirstHostPromptRequiresApproval(t *testing.T) {
	host, port, _ := testSSHServer(t)
	knownFile := filepath.Join(t.TempDir(), "ssh_known_hosts")
	spec, err := sshCommand(host, port)
	if err != nil {
		t.Fatal(err)
	}
	spec.args = append([]string{
		"-F", "none",
		"-o", "UserKnownHostsFile=" + knownFile,
		"-o", "GlobalKnownHostsFile=" + knownFile,
		"-o", "StrictHostKeyChecking=ask",
		"-o", "BatchMode=no",
		"-o", "UpdateHostKeys=no",
	}, spec.args...)
	m := NewManager(nil, nil)
	info, err := m.createSpec(spec, "SSH confirm test", "ssh", 90, 28)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, item := range m.List() {
			if item.ID == info.ID && item.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()
	waitSSHOutput(t, m, info.ID, "Are you sure you want to continue connecting")
	if _, err := os.Stat(knownFile); err == nil {
		t.Fatal("host key saved before user approval")
	}
	if err := m.Write(info.ID, "yes\r\n"); err != nil {
		t.Fatal(err)
	}
	waitSSHOutput(t, m, info.ID, "DIGGERTERM_SSH_WELCOME")
	data, err := os.ReadFile(knownFile)
	if err != nil || len(data) == 0 {
		t.Fatalf("host key not saved after approval: %v", err)
	}
}

func TestWindowsSSHHostKeyVerification(t *testing.T) {
	host, port, _ := testSSHServer(t)
	// A DIFFERENT key is deliberately trusted; connecting must fail closed.
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	other, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	knownFile := filepath.Join(t.TempDir(), "ssh_known_hosts")
	if err := os.WriteFile(knownFile, []byte(knownhosts.Line(
		[]string{fmt.Sprintf("[%s]:%d", host, port)}, other.PublicKey())+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec, err := sshCommand(host, port)
	if err != nil {
		t.Fatal(err)
	}
	spec.args = append([]string{
		"-F", "none",
		"-o", "UserKnownHostsFile=" + knownFile,
		"-o", "GlobalKnownHostsFile=" + knownFile,
		"-o", "StrictHostKeyChecking=yes",
		"-o", "BatchMode=yes",
		"-o", "UpdateHostKeys=no",
	}, spec.args...)
	m := NewManager(nil, nil)
	info, err := m.createSpec(spec, "SSH untrusted", "ssh", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, item := range m.List() {
			if item.Status == "running" {
				_ = m.Terminate(info.ID)
			}
		}
	}()
	waitSSHOutput(t, m, info.ID, "Host key verification failed")
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if m.List()[0].Status == "exited" {
			out, err := m.Snapshot(info.ID)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := base64.StdEncoding.DecodeString(out.Data)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "DIGGERTERM_SSH_WELCOME") {
				t.Fatal("SSH accepted a changed host key")
			}
			return
		}
		time.Sleep(60 * time.Millisecond)
	}
	t.Fatal("SSH did not reject the changed host key")
}
