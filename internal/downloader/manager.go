package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"curler/internal/config"
)

type TaskSnapshot struct {
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
}

type Manager struct {
	mu            sync.RWMutex
	Tasks         []*Task        `json:"tasks"`
	Config        *config.Config `json:"-"`
	UpdateCB      func()         `json:"-"`
	activeContext map[string]context.CancelFunc
}

func NewManager(cfg *config.Config, updateCB func()) *Manager {
	m := &Manager{
		Tasks:         make([]*Task, 0),
		Config:        cfg,
		UpdateCB:      updateCB,
		activeContext: make(map[string]context.CancelFunc),
	}
	m.LoadState()
	return m
}

func (m *Manager) SaveState() {
	m.mu.RLock()
	tasksCopy := make([]*Task, len(m.Tasks))
	copy(tasksCopy, m.Tasks)
	m.mu.RUnlock()

	snapshots := make([]TaskSnapshot, 0, len(tasksCopy))
	for _, t := range tasksCopy {
		logsCopy := t.GetLogs()

		t.mu.RLock()
		snap := TaskSnapshot{
			ID:              t.ID,
			URL:             t.URL,
			Filename:        t.Filename,
			SaveDir:         t.SaveDir,
			Status:          t.Status,
			TotalBytes:      t.TotalBytes,
			DownloadedBytes: t.DownloadedBytes,
			Speed:           t.Speed,
			ETA:             t.ETA,
			Progress:        t.Progress,
			ErrorMsg:        t.ErrorMsg,
			CurlCmd:         t.CurlCmd,
			Logs:            logsCopy,
			SupportsRange:   t.SupportsRange,
			CreatedAt:       t.CreatedAt,
			CompletedAt:     t.CompletedAt,
		}
		t.mu.RUnlock()
		snapshots = append(snapshots, snap)
	}

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m.Config.StateFilePath, data, 0644)
}

func (m *Manager) LoadState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.Config.StateFilePath)
	if err != nil {
		return
	}

	var snapshots []TaskSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return
	}

	m.Tasks = make([]*Task, 0, len(snapshots))
	for _, s := range snapshots {
		status := s.Status
		if status == StatusDownloading {
			status = StatusPaused
		}
		t := &Task{
			ID:              s.ID,
			URL:             s.URL,
			Filename:        s.Filename,
			SaveDir:         s.SaveDir,
			Status:          status,
			TotalBytes:      s.TotalBytes,
			DownloadedBytes: s.DownloadedBytes,
			Speed:           s.Speed,
			ETA:             s.ETA,
			Progress:        s.Progress,
			ErrorMsg:        s.ErrorMsg,
			CurlCmd:         s.CurlCmd,
			Logs:            s.Logs,
			SupportsRange:   s.SupportsRange,
			CreatedAt:       s.CreatedAt,
			CompletedAt:     s.CompletedAt,
		}
		m.Tasks = append(m.Tasks, t)
	}
}

func (m *Manager) AddTask(rawURL, filename, saveDir string) (*Task, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	if saveDir == "" {
		saveDir = m.Config.DownloadDir
	}

	// Probe URL
	probe := ProbeURL(rawURL)

	if filename == "" {
		filename = probe.SuggestedName
	}

	// Create unique ID
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	task := &Task{
		ID:            id,
		URL:           rawURL,
		Filename:      filename,
		SaveDir:       saveDir,
		Status:        StatusPending,
		TotalBytes:    probe.TotalBytes,
		SupportsRange: probe.SupportsRange,
		CreatedAt:     time.Now(),
		Logs:          make([]string, 0),
	}

	m.mu.Lock()
	m.Tasks = append([]*Task{task}, m.Tasks...) // Add to top
	m.mu.Unlock()

	m.SaveState()
	m.StartTask(task.ID)

	return task, nil
}

func (m *Manager) StartTask(id string) {
	m.mu.Lock()
	var task *Task
	for _, t := range m.Tasks {
		if t.ID == id {
			task = t
			break
		}
	}
	if task == nil {
		m.mu.Unlock()
		return
	}

	// If already downloading, do nothing
	if task.GetStatus() == StatusDownloading {
		m.mu.Unlock()
		return
	}

	// Extract previous context cancel function to execute after dropping m.mu
	oldCancel := m.activeContext[id]

	ctx, cancel := context.WithCancel(context.Background())
	m.activeContext[id] = cancel
	m.mu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.activeContext, id)
			m.mu.Unlock()

			m.SaveState()

			status := task.GetStatus()
			if status == StatusCompleted {
				config.SendArchNotification("Download Completed", fmt.Sprintf("Finished downloading %s", task.Filename))
			} else if status == StatusError {
				config.SendArchNotification("Download Failed", fmt.Sprintf("Failed to download %s: %s", task.Filename, task.ErrorMsg))
			}
		}()

		_ = StartCurlTask(ctx, task, func() {
			if m.UpdateCB != nil {
				m.UpdateCB()
			}
		})
	}()
}

func (m *Manager) PauseTask(id string) {
	m.mu.Lock()
	cancel, ok := m.activeContext[id]
	if ok {
		delete(m.activeContext, id)
	}

	var targetTask *Task
	for _, t := range m.Tasks {
		if t.ID == id {
			targetTask = t
			break
		}
	}
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	} else if targetTask != nil {
		targetTask.mu.Lock()
		targetTask.Status = StatusPaused
		targetTask.Speed = 0
		targetTask.ETA = 0
		targetTask.mu.Unlock()
	}

	m.SaveState()
}

func (m *Manager) ResumeTask(id string) {
	m.StartTask(id)
}

func (m *Manager) DeleteTask(id string, removeFile bool) {
	m.PauseTask(id)

	m.mu.Lock()
	var newTasks []*Task
	var targetTask *Task

	for _, t := range m.Tasks {
		if t.ID == id {
			targetTask = t
		} else {
			newTasks = append(newTasks, t)
		}
	}
	m.Tasks = newTasks
	m.mu.Unlock()

	if removeFile && targetTask != nil {
		_ = os.Remove(targetTask.FullPath())
	}

	m.SaveState()
}

func (m *Manager) ClearCompleted() {
	m.mu.Lock()
	var newTasks []*Task
	for _, t := range m.Tasks {
		if t.GetStatus() != StatusCompleted {
			newTasks = append(newTasks, t)
		}
	}
	m.Tasks = newTasks
	m.mu.Unlock()

	m.SaveState()
}

func (m *Manager) GetTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Task, len(m.Tasks))
	copy(result, m.Tasks)
	return result
}

func (m *Manager) GetTask(id string) *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.Tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}
