package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSSHTargetValidation(t *testing.T) {
	for _, value := range []string{"prod", "dev-box", "host.example.com", "alice@server", "user.name@host_1", "127.0.0.1"} {
		if err := validateSSHTarget(value); err != nil {
			t.Errorf("rejected %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "-oProxyCommand=x", "alice@-p", "@host", "user@", "user@@host", "user name@host", "host\n-oFoo=bar", "host:2222", "ssh://host", "a/b", ".ssh", "user;whoami", strings.Repeat("x", 256)} {
		if err := validateSSHTarget(value); err == nil {
			t.Errorf("accepted unsafe target %q", value)
		}
	}
}

func TestSSHCommandConstruction(t *testing.T) {
	for _, tc := range []struct {
		target string
		port   int
		args   []string
	}{
		{target: "dev", port: 0, args: []string{"-tt", "dev"}},
		{target: "alice@server.example", port: 2222, args: []string{"-tt", "-p", "2222", "alice@server.example"}},
	} {
		spec, err := sshCommand(tc.target, tc.port)
		if err != nil {
			t.Fatalf("sshCommand %q: %v", tc.target, err)
		}
		if spec.path == "" || !reflect.DeepEqual(spec.args, tc.args) {
			t.Fatalf("sshCommand result: %#v", spec)
		}
	}
	for _, badPort := range []int{-1, 65536, 999999} {
		if _, err := sshCommand("host", badPort); err == nil {
			t.Errorf("accepted port %d", badPort)
		}
	}
	if _, err := NewManager(nil, nil).CreateSSH("-oStrictHostKeyChecking=no", 22, 80, 24); err == nil {
		t.Fatal("option injection should not start a session")
	}
	if _, err := NewManager(nil, nil).CreateSSH("host", 0, 0, 24); err == nil {
		t.Fatal("invalid PTY size should not start SSH")
	}
}

func TestSSHCommandPreservesConfiguredKeepalive(t *testing.T) {
	for _, interval := range []string{"37", "0"} {
		t.Run("interval_"+interval, func(t *testing.T) {
			spec, err := sshCommand("diggerterm-test-alias", 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, arg := range spec.args {
				if strings.Contains(strings.ToLower(arg), "serveralive") {
					t.Fatalf("sshCommand overrides ssh_config keepalive: %q", arg)
				}
			}

			config := filepath.Join(t.TempDir(), "ssh_config")
			contents := "Host diggerterm-test-alias\n  ServerAliveInterval " + interval + "\n  ServerAliveCountMax 5\n"
			if err := os.WriteFile(config, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}

			args := append([]string{"-G", "-F", config}, spec.args...)
			output, err := exec.Command(spec.path, args...).Output()
			if err != nil {
				t.Fatalf("OpenSSH config evaluation: %v", err)
			}
			values := make(map[string]string)
			for _, line := range strings.Split(string(output), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 {
					values[fields[0]] = fields[1]
				}
			}
			if got := values["serveraliveinterval"]; got != interval {
				t.Fatalf("effective ServerAliveInterval = %q, want %q", got, interval)
			}
			if got, want := values["serveralivecountmax"], strconv.Itoa(5); got != want {
				t.Fatalf("effective ServerAliveCountMax = %q, want %q", got, want)
			}
		})
	}
}
