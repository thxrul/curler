package downloader

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProbeResult struct {
	TotalBytes    int64
	SuggestedName string
	SupportsRange bool
	StatusCode    int
}

// ProbeURL queries the remote URL to check content size, filename, and byte range support
func ProbeURL(targetURL string) ProbeResult {
	result := ProbeResult{
		SuggestedName: ExtractFilenameFromURL(targetURL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", targetURL, nil)
	if err != nil {
		return result
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; Curler/1.0)")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.ContentLength > 0 {
		result.TotalBytes = resp.ContentLength
	}

	if acceptRanges := resp.Header.Get("Accept-Ranges"); strings.EqualFold(acceptRanges, "bytes") {
		result.SupportsRange = true
	}

	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if idx := strings.Index(cd, "filename="); idx != -1 {
			name := cd[idx+9:]
			name = strings.Trim(name, `";' `)
			if name != "" {
				result.SuggestedName = name
			}
		}
	}

	return result
}

func ExtractFilenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Path == "" {
		return "downloaded_file"
	}
	base := filepath.Base(parsed.Path)
	if base == "" || base == "/" || base == "." {
		return "downloaded_file"
	}
	return base
}

// StartCurlTask launches curl with built-in -C - continuous resume flag
func StartCurlTask(ctx context.Context, task *Task, updateCb func()) error {
	task.mu.Lock()
	task.Status = StatusDownloading
	task.ErrorMsg = ""
	task.mu.Unlock()

	safeUpdate := func() {
		if updateCb != nil {
			updateCb()
		}
	}

	// Ensure save directory exists
	if err := os.MkdirAll(task.SaveDir, 0755); err != nil {
		task.mu.Lock()
		task.Status = StatusError
		task.ErrorMsg = fmt.Sprintf("Failed to create save dir: %v", err)
		task.mu.Unlock()
		safeUpdate()
		return err
	}

	targetPath := task.FullPath()

	// Get initial file size before starting curl (to confirm resume offset)
	var initialBytes int64 = 0
	if fi, err := os.Stat(targetPath); err == nil {
		initialBytes = fi.Size()
	}

	task.AddLog(fmt.Sprintf("[%s] Starting download via curl. Resuming from %s...",
		time.Now().Format("15:04:05"), FormatBytes(initialBytes)))

	// Build curl command line:
	// curl -L -C - --fail --show-error -o <targetPath> <URL>
	args := []string{
		"-L",
		"-C", "-",
		"--fail",
		"--show-error",
		"--user-agent", "Mozilla/5.0 (X11; Linux x86_64; Curler/1.0)",
		"-o", targetPath,
		task.URL,
	}

	task.CurlCmd = fmt.Sprintf("curl %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "curl", args...)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		task.mu.Lock()
		task.Status = StatusError
		task.ErrorMsg = fmt.Sprintf("Failed to pipe stderr: %v", err)
		task.mu.Unlock()
		safeUpdate()
		return err
	}

	task.mu.Lock()
	task.cmd = cmd
	task.mu.Unlock()

	if err := cmd.Start(); err != nil {
		task.mu.Lock()
		task.Status = StatusError
		task.ErrorMsg = fmt.Sprintf("Failed to start curl: %v", err)
		task.mu.Unlock()
		safeUpdate()
		return err
	}

	// Capture stderr output in background goroutine
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				task.AddLog(line)
			}
		}
	}()

	// Monitor file size and progress in background loop
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- cmd.Wait()
	}()

	var lastBytes int64 = initialBytes
	var lastTime time.Time = time.Now()

	for {
		select {
		case <-ctx.Done():
			// Context canceled (paused or interrupted)
			task.mu.Lock()
			task.Status = StatusPaused
			task.Speed = 0
			task.ETA = 0
			task.mu.Unlock()
			task.AddLog(fmt.Sprintf("[%s] Download paused/interrupted.", time.Now().Format("15:04:05")))
			safeUpdate()
			return nil

		case err := <-doneChan:
			// Process completed
			fi, statErr := os.Stat(targetPath)
			currBytes := int64(0)
			if statErr == nil {
				currBytes = fi.Size()
			}

			task.mu.Lock()
			task.DownloadedBytes = currBytes
			if task.TotalBytes > 0 && currBytes > 0 {
				task.Progress = (float64(currBytes) / float64(task.TotalBytes)) * 100
				if task.Progress > 100 {
					task.Progress = 100
				}
			}

			if err == nil {
				task.Status = StatusCompleted
				task.Progress = 100.0
				task.CompletedAt = time.Now()
				task.Speed = 0
				task.ETA = 0
				if task.TotalBytes <= 0 {
					task.TotalBytes = currBytes
				}
				task.AddLog(fmt.Sprintf("[%s] Download completed successfully! Total size: %s",
					time.Now().Format("15:04:05"), FormatBytes(currBytes)))
			} else {
				// If error was triggered by context cancellation, treat as paused
				if ctx.Err() != nil {
					task.Status = StatusPaused
					task.AddLog(fmt.Sprintf("[%s] Download paused.", time.Now().Format("15:04:05")))
				} else {
					task.Status = StatusError
					task.ErrorMsg = fmt.Sprintf("curl exited with error: %v", err)
					task.AddLog(fmt.Sprintf("[%s] Error: %v", time.Now().Format("15:04:05"), err))
				}
				task.Speed = 0
				task.ETA = 0
			}
			task.mu.Unlock()
			safeUpdate()
			return err

		case <-ticker.C:
			// Update metrics
			fi, err := os.Stat(targetPath)
			if err == nil {
				currBytes := fi.Size()
				now := time.Now()
				durationSec := now.Sub(lastTime).Seconds()

				var currentSpeed int64 = 0
				if durationSec > 0 {
					delta := currBytes - lastBytes
					if delta < 0 {
						delta = 0
					}
					currentSpeed = int64(float64(delta) / durationSec)
				}

				task.mu.Lock()
				task.DownloadedBytes = currBytes
				task.Speed = currentSpeed

				if task.TotalBytes > 0 {
					task.Progress = (float64(currBytes) / float64(task.TotalBytes)) * 100
					if task.Progress > 100 {
						task.Progress = 100
					}

					remainingBytes := task.TotalBytes - currBytes
					if currentSpeed > 0 && remainingBytes > 0 {
						task.ETA = time.Duration(remainingBytes/currentSpeed) * time.Second
					} else {
						task.ETA = 0
					}
				}
				task.mu.Unlock()

				lastBytes = currBytes
				lastTime = now
				safeUpdate()
			}
		}
	}
}

// ParseTotalSizeFromLogs tries to parse Content-Length from curl output if HEAD probe failed
func ParseTotalSizeFromLogs(line string) int64 {
	lineLower := strings.ToLower(line)
	if idx := strings.Index(lineLower, "content-length:"); idx != -1 {
		val := strings.TrimSpace(line[idx+15:])
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			return size
		}
	}
	return 0
}
