package downloader

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type TaskStatus string

const (
	StatusPending     TaskStatus = "Pending"
	StatusDownloading TaskStatus = "Downloading"
	StatusPaused      TaskStatus = "Paused"
	StatusCompleted   TaskStatus = "Completed"
	StatusError       TaskStatus = "Error"
)

type Task struct {
	ID              string        `json:"id"`
	URL             string        `json:"url"`
	Filename        string        `json:"filename"`
	SaveDir         string        `json:"save_dir"`
	Status          TaskStatus    `json:"status"`
	TotalBytes      int64         `json:"total_bytes"`
	DownloadedBytes int64         `json:"downloaded_bytes"`
	Speed           int64         `json:"speed"`
	ETA             time.Duration `json:"eta"`
	Progress        float64       `json:"progress"`
	ErrorMsg        string        `json:"error_msg"`
	CurlCmd         string        `json:"curl_cmd"`
	Logs            []string      `json:"logs"`
	SupportsRange   bool          `json:"supports_range"`
	CreatedAt       time.Time     `json:"created_at"`
	CompletedAt     time.Time     `json:"completed_at"`

	mu     sync.RWMutex       `json:"-"`
	logMu  sync.Mutex         `json:"-"`
	cancel context.CancelFunc `json:"-"`
	cmd    *exec.Cmd          `json:"-"`
}

func (t *Task) FullPath() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return filepath.Join(t.SaveDir, t.Filename)
}

func (t *Task) AddLog(msg string) {
	t.logMu.Lock()
	defer t.logMu.Unlock()
	if len(t.Logs) > 200 {
		t.Logs = t.Logs[len(t.Logs)-200:]
	}
	t.Logs = append(t.Logs, msg)
}

func (t *Task) GetLogs() []string {
	t.logMu.Lock()
	defer t.logMu.Unlock()
	logsCopy := make([]string, len(t.Logs))
	copy(logsCopy, t.Logs)
	return logsCopy
}

func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

func (t *Task) GetProgress() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Progress
}

func (t *Task) GetDownloadedBytes() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.DownloadedBytes
}

func (t *Task) GetTotalBytes() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.TotalBytes
}

func (t *Task) GetSpeed() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Speed
}

func (t *Task) GetETA() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ETA
}

func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "--:--"
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
	}
	return fmt.Sprintf("%02dm %02ds", m, s)
}
