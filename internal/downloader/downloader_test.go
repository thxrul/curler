package downloader

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"curler/internal/config"
)

func TestContinuousDownloadAndResume(t *testing.T) {
	// Create a mock HTTP server that supports Byte-Ranges (HTTP 206)
	data := []byte("Hello, this is a test payload for continuous curl downloading in Arch Linux!")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int
			_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
			if err != nil {
				_, err = fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
				end = len(data) - 1
			}
			if start < len(data) {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(data[start : end+1])
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	cfg := ConfigForTest(tempDir)

	mgr := NewManager(cfg, nil)

	// Step 1: Add task
	task, err := mgr.AddTask(server.URL+"/testfile.txt", "testfile.txt", tempDir)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Wait for download to finish
	time.Sleep(1 * time.Second)

	tasks := mgr.GetTasks()
	if len(tasks) == 0 {
		t.Fatalf("Expected 1 task in manager")
	}

	filePath := filepath.Join(tempDir, "testfile.txt")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist at %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != string(data) {
		t.Fatalf("Downloaded content mismatch. Expected '%s', got '%s'", string(data), string(content))
	}

	// Step 2: Test Continuous Resume (-C -)
	// Truncate file to partial size (e.g. 10 bytes)
	partialLen := 10
	err = os.WriteFile(filePath, data[:partialLen], 0644)
	if err != nil {
		t.Fatalf("Failed to write partial file: %v", err)
	}

	// Reset status to paused and resume
	mgr.PauseTask(task.ID)
	mgr.ResumeTask(task.ID)

	time.Sleep(1 * time.Second)

	resumedContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read resumed file: %v", err)
	}

	if string(resumedContent) != string(data) {
		t.Fatalf("Resumed file content mismatch. Expected '%s', got '%s'", string(data), string(resumedContent))
	}
}

func ConfigForTest(dir string) *config.Config {
	return &config.Config{
		DownloadDir:   dir,
		MaxConcurrent: 2,
		NotifyEnabled: false,
		StateFilePath: filepath.Join(dir, "state.json"),
	}
}
