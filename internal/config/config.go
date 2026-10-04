package config

import (
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	DownloadDir   string `json:"download_dir"`
	MaxConcurrent int    `json:"max_concurrent"`
	NotifyEnabled bool   `json:"notify_enabled"`
	StateFilePath string `json:"-"`
}

func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		_ = os.MkdirAll(downloadsDir, 0755)
	}

	configDir := filepath.Join(homeDir, ".config", "curler")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		_ = os.MkdirAll(configDir, 0755)
	}

	stateFile := filepath.Join(configDir, "state.json")

	// Check if notify-send (Arch / libnotify) is installed
	_, notifyErr := exec.LookPath("notify-send")

	return &Config{
		DownloadDir:   downloadsDir,
		MaxConcurrent: 3,
		NotifyEnabled: notifyErr == nil,
		StateFilePath: stateFile,
	}
}

// SendArchNotification triggers desktop notifications if notify-send is present
func SendArchNotification(title, message string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		_ = exec.Command("notify-send", "-a", "Curler", "-i", "download", title, message).Run()
	}
}

// CheckCurlAvailable ensures curl binary exists on the system
func CheckCurlAvailable() (string, error) {
	path, err := exec.LookPath("curl")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return path, nil
	}
	// Return first line of curl version
	lines := string(out)
	for i, ch := range lines {
		if ch == '\n' {
			return lines[:i], nil
		}
	}
	return lines, nil
}
