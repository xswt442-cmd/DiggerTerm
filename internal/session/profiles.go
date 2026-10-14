package session

import (
	"os"
	"os/exec"
	"runtime"
)

type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type commandSpec struct {
	path string
	args []string
	dir  string
}

// Profiles only lists commands supported by the current host. WSL is hosted
// through Windows ConPTY; it is not a separate process owner.
func Profiles() []Profile {
	if runtime.GOOS == "windows" {
		result := []Profile{
			{ID: "powershell", Name: "PowerShell", Description: "Windows PowerShell"},
			{ID: "cmd", Name: "Command Prompt", Description: "Windows CMD"},
		}
		if _, err := exec.LookPath("wsl.exe"); err == nil {
			result = append(result, Profile{ID: "wsl", Name: "WSL", Description: "Default Linux distribution"})
		}
		if _, err := exec.LookPath("ssh.exe"); err == nil {
			result = append(result, Profile{ID: "ssh", Name: "SSH", Description: "OpenSSH remote terminal"})
		}
		return result
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	result := []Profile{{ID: "shell", Name: "Local Shell", Description: shell}}
	if _, err := exec.LookPath("ssh"); err == nil {
		result = append(result, Profile{ID: "ssh", Name: "SSH", Description: "OpenSSH remote terminal"})
	}
	return result
}

func profileCommand(id string) (commandSpec, bool) {
	switch id {
	case "powershell":
		if runtime.GOOS == "windows" {
			return commandSpec{path: "powershell.exe", args: []string{"-NoLogo"}}, true
		}
	case "cmd":
		if runtime.GOOS == "windows" {
			return commandSpec{path: "cmd.exe", args: []string{"/Q"}}, true
		}
	case "wsl":
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("wsl.exe"); err == nil {
				return commandSpec{path: "wsl.exe"}, true
			}
		}
	case "shell":
		if runtime.GOOS != "windows" {
			path := os.Getenv("SHELL")
			if path == "" {
				path = "/bin/sh"
			}
			return commandSpec{path: path}, true
		}
	}
	return commandSpec{}, false
}
