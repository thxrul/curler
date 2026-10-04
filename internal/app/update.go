package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"curler/internal/downloader"
)

func (m Model) FilteredTasks() []*downloader.Task {
	allTasks := m.Manager.GetTasks()
	if m.ActiveTab == 0 {
		return allTasks
	}

	var filtered []*downloader.Task
	for _, t := range allTasks {
		status := t.GetStatus()
		switch m.ActiveTab {
		case 1: // Downloading
			if status == downloader.StatusDownloading || status == downloader.StatusPending {
				filtered = append(filtered, t)
			}
		case 2: // Completed
			if status == downloader.StatusCompleted {
				filtered = append(filtered, t)
			}
		case 3: // Paused/Error
			if status == downloader.StatusPaused || status == downloader.StatusError {
				filtered = append(filtered, t)
			}
		}
	}
	return filtered
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - 8
		m.LogViewport.Width = msg.Width - 10
		m.LogViewport.Height = 15

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Handle key based on view mode
		switch m.ViewMode {
		case ViewMain:
			return m.handleMainKeys(msg)
		case ViewNewDownload:
			return m.handleNewDownloadKeys(msg)
		case ViewDetails:
			return m.handleDetailsKeys(msg)
		case ViewHelp:
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "?" {
				m.ViewMode = ViewMain
			}
			return m, nil
		case ViewConfirmDelete:
			return m.handleConfirmDeleteKeys(msg)
		}

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case UpdateStateMsg:
		// Refresh view
		return m, nil
	}

	// Update text inputs if in new download mode
	if m.ViewMode == ViewNewDownload {
		var cmd tea.Cmd
		if m.FocusIndex == 0 {
			m.URLInput, cmd = m.URLInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.FocusIndex == 1 {
			m.FilenameInput, cmd = m.FilenameInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.FocusIndex == 2 {
			m.SaveDirInput, cmd = m.SaveDirInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update log viewport if in details mode
	if m.ViewMode == ViewDetails {
		var cmd tea.Cmd
		m.LogViewport, cmd = m.LogViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tasks := m.FilteredTasks()

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "1", "2", "3", "4":
		tabIdx := int(msg.String()[0] - '1')
		m.ActiveTab = tabIdx
		m.SelectedIndex = 0

	case "tab":
		m.ActiveTab = (m.ActiveTab + 1) % 4
		m.SelectedIndex = 0

	case "shift+tab":
		m.ActiveTab = (m.ActiveTab + 3) % 4
		m.SelectedIndex = 0

	case "up", "k":
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}

	case "down", "j":
		if m.SelectedIndex < len(tasks)-1 {
			m.SelectedIndex++
		}

	case "home", "g":
		m.SelectedIndex = 0

	case "end", "G":
		if len(tasks) > 0 {
			m.SelectedIndex = len(tasks) - 1
		}

	case "n":
		m.ViewMode = ViewNewDownload
		m.URLInput.SetValue("")
		m.FilenameInput.SetValue("")
		m.URLInput.Focus()
		m.FocusIndex = 0

	case "space", "p":
		if len(tasks) > 0 && m.SelectedIndex < len(tasks) {
			task := tasks[m.SelectedIndex]
			if task.GetStatus() == downloader.StatusDownloading {
				m.Manager.PauseTask(task.ID)
			} else {
				m.Manager.ResumeTask(task.ID)
			}
		}

	case "enter", "l":
		if len(tasks) > 0 && m.SelectedIndex < len(tasks) {
			task := tasks[m.SelectedIndex]
			m.ViewMode = ViewDetails
			logText := strings.Join(task.GetLogs(), "\n")
			if logText == "" {
				logText = "No curl output log recorded yet."
			}
			m.LogViewport.SetContent(logText)
			m.LogViewport.GotoBottom()
		}

	case "d", "delete":
		if len(tasks) > 0 && m.SelectedIndex < len(tasks) {
			task := tasks[m.SelectedIndex]
			m.DeleteTaskID = task.ID
			m.DeleteTaskName = task.Filename
			m.RemoveFileCheck = false
			m.ViewMode = ViewConfirmDelete
		}

	case "c":
		m.Manager.ClearCompleted()
		m.SelectedIndex = 0

	case "?":
		m.ViewMode = ViewHelp

	case "r":
		if len(tasks) > 0 && m.SelectedIndex < len(tasks) {
			task := tasks[m.SelectedIndex]
			m.Manager.ResumeTask(task.ID)
		}
	}

	return m, nil
}

func (m Model) handleNewDownloadKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.ViewMode = ViewMain
		return m, nil

	case "tab", "down":
		m.FocusIndex = (m.FocusIndex + 1) % 5
		m.updateFocus()
		return m, nil

	case "shift+tab", "up":
		m.FocusIndex = (m.FocusIndex + 4) % 5
		m.updateFocus()
		return m, nil

	case "enter":
		if m.FocusIndex == 4 { // Cancel
			m.ViewMode = ViewMain
			return m, nil
		}

		// Submit download
		urlVal := strings.TrimSpace(m.URLInput.Value())
		if urlVal == "" {
			m.BannerMsg = "Error: URL cannot be empty!"
			return m, nil
		}

		filenameVal := strings.TrimSpace(m.FilenameInput.Value())
		saveDirVal := strings.TrimSpace(m.SaveDirInput.Value())

		_, err := m.Manager.AddTask(urlVal, filenameVal, saveDirVal)
		if err != nil {
			m.BannerMsg = err.Error()
		} else {
			m.BannerMsg = "Download started!"
			m.ViewMode = ViewMain
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) updateFocus() {
	m.URLInput.Blur()
	m.FilenameInput.Blur()
	m.SaveDirInput.Blur()

	switch m.FocusIndex {
	case 0:
		m.URLInput.Focus()
	case 1:
		m.FilenameInput.Focus()
	case 2:
		m.SaveDirInput.Focus()
	}
}

func (m Model) handleDetailsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "l", "enter":
		m.ViewMode = ViewMain
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "q":
		m.ViewMode = ViewMain
		return m, nil

	case "space", "f":
		m.RemoveFileCheck = !m.RemoveFileCheck
		return m, nil

	case "enter", "y":
		m.Manager.DeleteTask(m.DeleteTaskID, m.RemoveFileCheck)
		m.ViewMode = ViewMain
		if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Check Tab hit regions
			if m.TabHitRegions != nil {
				for _, r := range *m.TabHitRegions {
					if msg.X >= r.X1 && msg.X <= r.X2 && msg.Y >= r.Y1 && msg.Y <= r.Y2 {
						m.ActiveTab = r.Index
						m.SelectedIndex = 0
						return m, nil
					}
				}
			}

			// Check Row hit regions
			if m.RowHitRegions != nil {
				for _, r := range *m.RowHitRegions {
					if msg.X >= r.X1 && msg.X <= r.X2 && msg.Y >= r.Y1 && msg.Y <= r.Y2 {
						m.SelectedIndex = r.Index
						return m, nil
					}
				}
			}

			// Check Action Button hit regions
			if m.ButtonHitRegions != nil {
				for _, r := range *m.ButtonHitRegions {
					if msg.X >= r.X1 && msg.X <= r.X2 && msg.Y >= r.Y1 && msg.Y <= r.Y2 {
						switch r.Action {
						case "new":
							m.ViewMode = ViewNewDownload
							m.URLInput.SetValue("")
							m.FilenameInput.SetValue("")
							m.URLInput.Focus()
							m.FocusIndex = 0
						case "clear":
							m.Manager.ClearCompleted()
						case "help":
							m.ViewMode = ViewHelp
						case "pause_resume":
							tasks := m.FilteredTasks()
							if r.Index >= 0 && r.Index < len(tasks) {
								t := tasks[r.Index]
								if t.GetStatus() == downloader.StatusDownloading {
									m.Manager.PauseTask(t.ID)
								} else {
									m.Manager.ResumeTask(t.ID)
								}
							}
						case "details":
							tasks := m.FilteredTasks()
							if r.Index >= 0 && r.Index < len(tasks) {
								t := tasks[r.Index]
								m.ViewMode = ViewDetails
								logText := strings.Join(t.GetLogs(), "\n")
								m.LogViewport.SetContent(logText)
								m.LogViewport.GotoBottom()
							}
						case "delete":
							tasks := m.FilteredTasks()
							if r.Index >= 0 && r.Index < len(tasks) {
								t := tasks[r.Index]
								m.DeleteTaskID = t.ID
								m.DeleteTaskName = t.Filename
								m.RemoveFileCheck = false
								m.ViewMode = ViewConfirmDelete
							}
						}
						return m, nil
					}
				}
			}
		}

	case tea.MouseActionMotion:
		// Scroll support
	}

	// Mouse Wheel Scrolling
	if msg.Button == tea.MouseButtonWheelUp {
		if m.ViewMode == ViewDetails {
			m.LogViewport.LineUp(2)
		} else if m.SelectedIndex > 0 {
			m.SelectedIndex--
		}
	} else if msg.Button == tea.MouseButtonWheelDown {
		tasks := m.FilteredTasks()
		if m.ViewMode == ViewDetails {
			m.LogViewport.LineDown(2)
		} else if m.SelectedIndex < len(tasks)-1 {
			m.SelectedIndex++
		}
	}

	return m, nil
}
