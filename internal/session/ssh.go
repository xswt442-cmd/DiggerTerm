package session

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// sshCommand intentionally delegates user configuration, identities,
// host-key verification, and interactive authentication to OpenSSH.
// Never add options disabling StrictHostKeyChecking or using shell commands.
func sshCommand(target string, port int) (commandSpec, error) {
	if err := validateSSHTarget(target); err != nil {
		return commandSpec{}, err
	}
	if port < 0 || port > 65535 {
		return commandSpec{}, fmt.Errorf("SSH port must be 1-65535 (or 0 to use ssh_config)")
	}
	program := "ssh"
	if runtime.GOOS == "windows" {
		program = "ssh.exe"
	}
	path, err := exec.LookPath(program)
	if err != nil {
		return commandSpec{}, fmt.Errorf("OpenSSH client not found on PATH: %w", err)
	}
	args := []string{"-tt"} // remote interactive PTY, even if host config disables RequestTTY
	if port != 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, target)
	return commandSpec{path: path, args: args}, nil
}

// SSH aliases and user@host destinations are supported. Host aliases in
// ~/.ssh/config can provide IPv6 addresses, nonstandard ports, keys and
// ProxyJump settings. Do not pass SSH option strings from user input.
func validateSSHTarget(target string) error {
	if target == "" || len(target) > 255 || strings.Count(target, "@") > 1 {
		return fmt.Errorf("enter an SSH host alias or user@hostname (1-255 ASCII characters)")
	}
	for _, token := range strings.Split(target, "@") {
		if token == "" || token[0] == '-' || token[0] == '.' {
			return fmt.Errorf("invalid SSH host or username")
		}
		for _, c := range token {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
				return fmt.Errorf("SSH destination supports only hostname aliases or user@hostname; configure IPv6/advanced options in ~/.ssh/config")
			}
		}
	}
	return nil
}
