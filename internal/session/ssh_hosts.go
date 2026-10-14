package session

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxSSHConfigBytes = 512 * 1024
const maxSSHHostSuggestions = 100

// SSHHostAliases returns only literal, non-pattern Host names from the user's
// personal OpenSSH config. It never resolves HostName, IdentityFile,
// ProxyCommand or Include paths; no credentials are returned or logged.
func SSHHostAliases() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locate SSH config: %w", err)
	}
	path := filepath.Join(home, ".ssh", "config")
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check SSH config: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxSSHConfigBytes {
		return nil, errors.New("SSH config is not a small regular file")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read SSH config: %w", err)
	}
	return parseSSHHostAliases(content), nil
}

func parseSSHHostAliases(content []byte) []string {
	suggestions := make([]string, 0)
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 1024), 16*1024)
	for scanner.Scan() {
		// Supports the common "Host alias1 alias2" and "Host=alias" forms.
		// Quoted and wildcard patterns are deliberately skipped.
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		line = strings.ReplaceAll(line, "=", " = ")
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Host") {
			continue
		}
		aliases := fields[1:]
		if aliases[0] == "=" {
			aliases = aliases[1:]
		}
		for _, name := range aliases {
			if name == "=" || strings.ContainsAny(name, "*?!\"'") || strings.Contains(name, "@") || validateSSHTarget(name) != nil {
				continue
			}
			key := strings.ToLower(name)
			if !seen[key] {
				seen[key] = true
				suggestions = append(suggestions, name)
			}
		}
	}
	sort.Slice(suggestions, func(i, j int) bool {
		return strings.ToLower(suggestions[i]) < strings.ToLower(suggestions[j])
	})
	if len(suggestions) > maxSSHHostSuggestions {
		return suggestions[:maxSSHHostSuggestions]
	}
	return suggestions
}
