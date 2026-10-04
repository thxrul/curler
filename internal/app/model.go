package app

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"curler/internal/config"
	"curler/internal/downloader"
)

type ViewMode int

const (
	ViewMain ViewMode = iota
	ViewNewDownload
	ViewDetails
	ViewHelp
	ViewConfirmDelete
)

type Model struct {
	Manager     *downloader.Manager
	Config      *config.Config
	CurlVersion string

	ViewMode  ViewMode
	ActiveTab int // 0: All, 1: Downloading, 2: Completed, 3: Paused/Failed

	SelectedIndex int
	Viewport      viewport.Model

	// Inputs for New Download Modal
	URLInput      textinput.Model
	FilenameInput textinput.Model
	SaveDirInput  textinput.Model
	FocusIndex    int // 0: URL, 1: Filename, 2: SaveDir, 3: Submit, 4: Cancel

	// Details / Logs Modal
	LogViewport viewport.Model

	// Delete Confirmation state
	DeleteTaskID    string
	DeleteTaskName  string
	RemoveFileCheck bool

	// UI Dimensions
	Width  int
	Height int

	// Error / Notification Banner
	BannerMsg string

	// Mouse click hit-regions stored during view render
	TabHitRegions    *[]HitRegion
	RowHitRegions    *[]HitRegion
	ButtonHitRegions *[]HitRegion
}

type HitRegion struct {
	X1, Y1, X2, Y2 int
	Action         string
	Index          int
	ID             string
}

type TickMsg struct{}
type UpdateStateMsg struct{}

func NewModel(cfg *config.Config) Model {
	curlVer, _ := config.CheckCurlAvailable()

	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/file.iso or http://..."
	urlInput.Focus()
	urlInput.CharLimit = 2048

	filenameInput := textinput.New()
	filenameInput.Placeholder = "Auto-detected if left empty"
	filenameInput.CharLimit = 255

	saveDirInput := textinput.New()
	saveDirInput.Placeholder = cfg.DownloadDir
	saveDirInput.SetValue(cfg.DownloadDir)
	saveDirInput.CharLimit = 1024

	vp := viewport.New(80, 20)
	logVp := viewport.New(80, 15)

	tabHits := make([]HitRegion, 0)
	rowHits := make([]HitRegion, 0)
	btnHits := make([]HitRegion, 0)

	m := Model{
		Config:           cfg,
		CurlVersion:      curlVer,
		ViewMode:         ViewMain,
		ActiveTab:        0,
		SelectedIndex:    0,
		Viewport:         vp,
		LogViewport:      logVp,
		URLInput:         urlInput,
		FilenameInput:    filenameInput,
		SaveDirInput:     saveDirInput,
		FocusIndex:       0,
		TabHitRegions:    &tabHits,
		RowHitRegions:    &rowHits,
		ButtonHitRegions: &btnHits,
	}

	m.Manager = downloader.NewManager(cfg, func() {
		// Callback handled via Tea program message channel
	})

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
	)
}
