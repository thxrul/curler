package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"curler/internal/downloader"
)

var (
	// Minimalist Color Palette
	AccentCyan  = lipgloss.Color("#38bdf8") // Nordic Cyan
	SoftWhite   = lipgloss.Color("#f4f4f5")
	MutedText   = lipgloss.Color("#71717a")
	SubtleBg    = lipgloss.Color("#18181b")
	SelectionBg = lipgloss.Color("#27272a")
	BorderColor = lipgloss.Color("#3f3f46")

	StatusDown  = lipgloss.Color("#38bdf8") // Cyan
	StatusPause = lipgloss.Color("#fbbf24") // Amber
	StatusDone  = lipgloss.Color("#34d399") // Emerald
	StatusErr   = lipgloss.Color("#f87171") // Rose
	StatusWait  = lipgloss.Color("#a1a1aa") // Muted

	// ASCII Art Logo
	ASCIIArtLogo = `  ___ _   _ ___ _    _____ ___ 
 / __| | | | _ \ |  | __| _ \
| (__| |_| |   / |__| _||   /
 \___|\___/|_|_\____|___|_|_\`

	// Typography & Layout Styles
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentCyan)

	HeaderMetaStyle = lipgloss.NewStyle().
			Foreground(MutedText)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentCyan).
			Underline(true).
			Padding(0, 1)

	TabInactiveStyle = lipgloss.NewStyle().
			Foreground(MutedText).
			Padding(0, 1)

	TableHeadStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(MutedText).
			Padding(0, 1)

	RowNormalStyle = lipgloss.NewStyle().
			Foreground(SoftWhite).
			Padding(0, 1)

	RowSelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SoftWhite).
			Background(SelectionBg).
			Padding(0, 1)

	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentCyan).
			Padding(1, 2)

	FooterLegendStyle = lipgloss.NewStyle().
			Foreground(MutedText).
			Padding(0, 1)

	KeyBadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(AccentCyan)

	DescTextStyle = lipgloss.NewStyle().
			Foreground(MutedText)
)

func RenderAsciiLogo(width int) string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(AccentCyan).Render(ASCIIArtLogo)
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, logo)
}

func StatusBadge(status downloader.TaskStatus) string {
	switch status {
	case downloader.StatusDownloading:
		return lipgloss.NewStyle().Foreground(StatusDown).Render("● Downloading")
	case downloader.StatusPaused:
		return lipgloss.NewStyle().Foreground(StatusPause).Render("⏸ Paused")
	case downloader.StatusCompleted:
		return lipgloss.NewStyle().Foreground(StatusDone).Render("✓ Completed")
	case downloader.StatusError:
		return lipgloss.NewStyle().Foreground(StatusErr).Render("✕ Error")
	default:
		return lipgloss.NewStyle().Foreground(StatusWait).Render("○ Pending")
	}
}

func ProgressBar(percent float64, width int) string {
	if width < 5 {
		width = 5
	}

	filledLen := int((percent / 100.0) * float64(width))
	if filledLen > width {
		filledLen = width
	}
	if filledLen < 0 {
		filledLen = 0
	}

	emptyLen := width - filledLen

	filledStr := strings.Repeat("━", filledLen)
	emptyStr := strings.Repeat("─", emptyLen)

	filledStyle := lipgloss.NewStyle().Foreground(AccentCyan)
	emptyStyle := lipgloss.NewStyle().Foreground(BorderColor)

	percentStr := fmt.Sprintf("%5.1f%%", percent)
	percentStyle := lipgloss.NewStyle().Foreground(SoftWhite)

	return fmt.Sprintf("%s%s %s", filledStyle.Render(filledStr), emptyStyle.Render(emptyStr), percentStyle.Render(percentStr))
}

type KeyHelp struct {
	Key  string
	Desc string
}

func RenderHelpFooter(width int, keys []KeyHelp) string {
	var items []string
	for _, k := range keys {
		keyStr := KeyBadgeStyle.Render("[" + k.Key + "]")
		descStr := DescTextStyle.Render(k.Desc)
		items = append(items, fmt.Sprintf("%s %s", keyStr, descStr))
	}
	line := strings.Join(items, "  ·  ")
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, FooterLegendStyle.Render(line))
}
