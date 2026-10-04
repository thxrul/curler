package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"curler/internal/downloader"
	"curler/internal/ui"
)

func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing Curler..."
	}

	// Reset hit regions before rendering
	if m.TabHitRegions != nil {
		*m.TabHitRegions = make([]HitRegion, 0)
	}
	if m.RowHitRegions != nil {
		*m.RowHitRegions = make([]HitRegion, 0)
	}
	if m.ButtonHitRegions != nil {
		*m.ButtonHitRegions = make([]HitRegion, 0)
	}

	var body string

	switch m.ViewMode {
	case ViewNewDownload:
		body = m.renderMainContent()
		return m.overlayModal(body, m.renderNewDownloadModal())

	case ViewDetails:
		body = m.renderMainContent()
		return m.overlayModal(body, m.renderDetailsModal())

	case ViewHelp:
		body = m.renderMainContent()
		return m.overlayModal(body, m.renderHelpModal())

	case ViewConfirmDelete:
		body = m.renderMainContent()
		return m.overlayModal(body, m.renderConfirmDeleteModal())

	default:
		return m.renderMainContent()
	}
}

func (m Model) renderMainContent() string {
	allTasks := m.Manager.GetTasks()
	activeCount, pausedCount, doneCount := 0, 0, 0

	for _, t := range allTasks {
		status := t.GetStatus()
		switch status {
		case downloader.StatusDownloading, downloader.StatusPending:
			activeCount++
		case downloader.StatusPaused, downloader.StatusError:
			pausedCount++
		case downloader.StatusCompleted:
			doneCount++
		}
	}

	currentY := 0

	// 1. Centered ASCII Art Title
	asciiHeader := ui.RenderAsciiLogo(m.Width)
	currentY += lipgloss.Height(asciiHeader)

	// 2. Centered Subtitle Meta Summary (Dependency names removed!)
	metaText := fmt.Sprintf("%d active  ·  %d paused  ·  %d completed", activeCount, pausedCount, doneCount)
	metaLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, ui.HeaderMetaStyle.Render(metaText))
	currentY += lipgloss.Height(metaLine)

	// Centered Divider Line
	dividerWidth := 60
	if dividerWidth > m.Width-4 {
		dividerWidth = m.Width - 4
	}
	if dividerWidth < 10 {
		dividerWidth = 10
	}
	dividerLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, ui.HeaderMetaStyle.Render(strings.Repeat("─", dividerWidth)))
	currentY += lipgloss.Height(dividerLine)

	// 3. Centered Filter Tabs & Action Buttons
	tabNames := []string{
		fmt.Sprintf("1: All (%d)", len(allTasks)),
		fmt.Sprintf("2: Downloading (%d)", activeCount),
		fmt.Sprintf("3: Completed (%d)", doneCount),
		fmt.Sprintf("4: Paused (%d)", pausedCount),
	}

	var renderedTabs []string
	var tabWidths []int

	for i, name := range tabNames {
		style := ui.TabInactiveStyle
		if i == m.ActiveTab {
			style = ui.TabActiveStyle
		}
		renderedTab := style.Render(name)
		renderedTabs = append(renderedTabs, renderedTab)
		tabWidths = append(tabWidths, lipgloss.Width(renderedTab))
	}

	newBtnStr := ui.KeyBadgeStyle.Render("[+ New]")
	clearBtnStr := ui.DescTextStyle.Render("[Clear Done]")

	tabsRowStr := strings.Join(renderedTabs, "   ")
	totalTabWidth := lipgloss.Width(tabsRowStr)

	tabsStartX := (m.Width - totalTabWidth) / 2
	if tabsStartX < 0 {
		tabsStartX = 0
	}

	// Calculate tab hit regions accurately
	currX := tabsStartX
	for i, w := range tabWidths {
		if m.TabHitRegions != nil {
			*m.TabHitRegions = append(*m.TabHitRegions, HitRegion{
				X1:     currX,
				Y1:     currentY,
				X2:     currX + w - 1,
				Y2:     currentY,
				Action: "switch_tab",
				Index:  i,
			})
		}
		currX += w + 3 // 3 spaces separator
	}

	tabsLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, tabsRowStr)
	currentY += lipgloss.Height(tabsLine)

	// Top Action buttons row below tabs
	actionBtnsStr := lipgloss.JoinHorizontal(lipgloss.Top, newBtnStr, "     ", clearBtnStr)
	totalBtnWidth := lipgloss.Width(actionBtnsStr)
	btnStartX := (m.Width - totalBtnWidth) / 2
	if btnStartX < 0 {
		btnStartX = 0
	}

	if m.ButtonHitRegions != nil {
		*m.ButtonHitRegions = append(*m.ButtonHitRegions, HitRegion{
			X1: btnStartX, Y1: currentY, X2: btnStartX + lipgloss.Width(newBtnStr), Y2: currentY, Action: "new",
		})
		btnStartX += lipgloss.Width(newBtnStr) + 5
		*m.ButtonHitRegions = append(*m.ButtonHitRegions, HitRegion{
			X1: btnStartX, Y1: currentY, X2: btnStartX + lipgloss.Width(clearBtnStr), Y2: currentY, Action: "clear",
		})
	}

	actionLine := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, actionBtnsStr)
	currentY += lipgloss.Height(actionLine)

	// Banner Msg (if present)
	var bannerLine string
	if m.BannerMsg != "" {
		bannerLine = lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, ui.DescTextStyle.Render("  ℹ "+m.BannerMsg))
		currentY += lipgloss.Height(bannerLine)
	}

	// 4. Centered Download Table View
	filtered := m.FilteredTasks()

	tableHeadRaw := fmt.Sprintf(" %-2s %-15s %-25s %-22s %-14s %-12s %-10s %s",
		"", "STATUS", "FILENAME", "PROGRESS", "SIZE", "SPEED", "ETA", "ACTIONS")

	tableWidth := lipgloss.Width(tableHeadRaw)
	tableStartX := (m.Width - tableWidth) / 2
	if tableStartX < 0 {
		tableStartX = 0
	}

	tableHeader := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, ui.TableHeadStyle.Render(tableHeadRaw))
	currentY += lipgloss.Height(tableHeader)

	var rows []string
	availableRows := m.Height - currentY - 3
	if availableRows < 1 {
		availableRows = 1
	}

	if len(filtered) == 0 {
		emptyStr := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center,
			ui.DescTextStyle.Padding(2, 0).Render("No downloads to display. Press [n] to add a download."))
		rows = append(rows, emptyStr)
	} else {
		for i, task := range filtered {
			if i >= availableRows {
				break
			}

			cursor := " "
			rowStyle := ui.RowNormalStyle
			if i == m.SelectedIndex {
				cursor = "›"
				rowStyle = ui.RowSelectedStyle
			}

			status := task.GetStatus()
			downloadedBytes := task.GetDownloadedBytes()
			totalBytes := task.GetTotalBytes()
			speed := task.GetSpeed()
			eta := task.GetETA()
			progress := task.GetProgress()

			statusBadge := ui.StatusBadge(status)

			filename := task.Filename
			if len(filename) > 23 {
				filename = filename[:20] + "..."
			}

			progBar := ui.ProgressBar(progress, 14)

			sizeStr := fmt.Sprintf("%s / %s", downloader.FormatBytes(downloadedBytes), downloader.FormatBytes(totalBytes))
			if totalBytes <= 0 {
				sizeStr = downloader.FormatBytes(downloadedBytes)
			}

			speedStr := "--"
			if status == downloader.StatusDownloading && speed > 0 {
				speedStr = fmt.Sprintf("%s/s", downloader.FormatBytes(speed))
			}

			etaStr := downloader.FormatDuration(eta)
			if status == downloader.StatusCompleted {
				etaStr = "Done"
			} else if status == downloader.StatusPaused {
				etaStr = "Paused"
			}

			actionText := "[Pause]"
			if status != downloader.StatusDownloading {
				actionText = "[Resume]"
			}
			actionsStr := fmt.Sprintf("%s [Logs] [Del]", actionText)

			rawRow := fmt.Sprintf(" %-2s %-15s %-25s %-22s %-14s %-12s %-10s %s",
				cursor, statusBadge, filename, progBar, sizeStr, speedStr, etaStr, actionsStr)

			renderedRow := rowStyle.Render(rawRow)
			line := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, renderedRow)

			if m.RowHitRegions != nil {
				*m.RowHitRegions = append(*m.RowHitRegions, HitRegion{
					X1:     tableStartX,
					Y1:     currentY,
					X2:     tableStartX + tableWidth,
					Y2:     currentY + lipgloss.Height(line) - 1,
					Action: "select_row",
					Index:  i,
					ID:     task.ID,
				})
			}

			currentY += lipgloss.Height(line)
			rows = append(rows, line)
		}
	}

	tableBody := strings.Join(rows, "\n")

	// 5. Centered Quick Shortcuts Legend Footer
	footerKeys := []ui.KeyHelp{
		{Key: "n", Desc: "new"},
		{Key: "space", Desc: "pause/resume"},
		{Key: "l", Desc: "logs"},
		{Key: "d", Desc: "delete"},
		{Key: "c", Desc: "clear done"},
		{Key: "1-4", Desc: "filter"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}

	footerLegend := ui.RenderHelpFooter(m.Width, footerKeys)

	content := lipgloss.JoinVertical(lipgloss.Left,
		asciiHeader,
		metaLine,
		dividerLine,
		tabsLine,
		actionLine,
		bannerLine,
		tableHeader,
		tableBody,
	)

	// Calculate vertical padding to position bottom legend cleanly
	contentHeight := lipgloss.Height(content)
	footerHeight := lipgloss.Height(footerLegend)
	fillLines := m.Height - contentHeight - footerHeight
	if fillLines < 0 {
		fillLines = 0
	}

	return content + strings.Repeat("\n", fillLines) + footerLegend
}

func (m Model) overlayModal(bg, modal string) string {
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, modal)
}

func (m Model) renderNewDownloadModal() string {
	title := ui.AppTitleStyle.Render("New Download")

	urlLabel := ui.DescTextStyle.Render("URL:")
	urlField := m.URLInput.View()

	fnLabel := ui.DescTextStyle.Render("Filename (optional):")
	fnField := m.FilenameInput.View()

	dirLabel := ui.DescTextStyle.Render("Save Directory:")
	dirField := m.SaveDirInput.View()

	btnStartStyle := ui.TabInactiveStyle
	btnCancelStyle := ui.TabInactiveStyle

	if m.FocusIndex == 3 {
		btnStartStyle = ui.TabActiveStyle
	} else if m.FocusIndex == 4 {
		btnCancelStyle = ui.TabActiveStyle
	}

	btnStart := btnStartStyle.Render("Start [Enter]")
	btnCancel := btnCancelStyle.Render("Cancel [Esc]")

	btns := lipgloss.JoinHorizontal(lipgloss.Top, btnStart, "    ", btnCancel)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		urlLabel,
		urlField,
		"",
		fnLabel,
		fnField,
		"",
		dirLabel,
		dirField,
		"",
		btns,
	)

	return ui.ModalStyle.Width(65).Render(content)
}

func (m Model) renderDetailsModal() string {
	tasks := m.FilteredTasks()
	if m.SelectedIndex >= len(tasks) {
		return ui.ModalStyle.Render("No task selected")
	}

	task := tasks[m.SelectedIndex]

	title := ui.AppTitleStyle.Render("Task Details & Curl Output")

	info := fmt.Sprintf(`%s %s
%s %s
%s %s
%s %s`,
		ui.DescTextStyle.Render("File:"), task.Filename,
		ui.DescTextStyle.Render("Save Path:"), filepath.Join(task.SaveDir, task.Filename),
		ui.DescTextStyle.Render("URL:"), task.URL,
		ui.DescTextStyle.Render("Command:"), task.CurlCmd,
	)

	logsHeader := ui.DescTextStyle.Render("────── Logs ──────")
	logsView := m.LogViewport.View()

	closeBtn := ui.KeyBadgeStyle.Render("[ Close (Esc) ]")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		info,
		"",
		logsHeader,
		logsView,
		"",
		closeBtn,
	)

	return ui.ModalStyle.Width(80).Render(content)
}

func (m Model) renderHelpModal() string {
	title := ui.AppTitleStyle.Render("Quick Shortcuts Legend")

	helpText := `
  n          Add new download
  space / p  Pause / resume selected download
  l / enter  View task details & curl output logs
  d          Delete selected download
  c          Clear completed downloads
  1 - 4      Filter view (All, Downloading, Completed, Paused)
  tab        Cycle filter tabs
  j / k      Navigate download list up / down
  ?          Toggle help window
  esc        Close modal
  q          Quit
`

	closeBtn := ui.KeyBadgeStyle.Render("[ Close (Esc) ]")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		helpText,
		closeBtn,
	)

	return ui.ModalStyle.Width(65).Render(content)
}

func (m Model) renderConfirmDeleteModal() string {
	title := ui.AppTitleStyle.Render("Delete Download Task")

	prompt := fmt.Sprintf("Delete task '%s'?", m.DeleteTaskName)

	checkbox := "[ ] Remove downloaded file from disk (space)"
	if m.RemoveFileCheck {
		checkbox = "[✓] Remove downloaded file from disk (space)"
	}
	checkboxStr := ui.KeyBadgeStyle.Render(checkbox)

	btnDelete := ui.KeyBadgeStyle.Render("Delete [Enter]")
	btnCancel := ui.DescTextStyle.Render("Cancel [Esc]")

	btns := lipgloss.JoinHorizontal(lipgloss.Top, btnDelete, "    ", btnCancel)

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		prompt,
		"",
		checkboxStr,
		"",
		btns,
	)

	return ui.ModalStyle.Width(60).Render(content)
}
