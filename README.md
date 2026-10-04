# Curler

**Curler** is a modern Terminal User Interface (TUI) download manager engineered specifically for Arch-based Linux distributions. It leverages native `curl` functionality to provide **continuous download support**, auto-resuming interrupted downloads from exact byte offsets.

---

## Key Features

- **Continuous Downloading (`curl -C -`)**: Built-in resume functionality. Paused or network-interrupted downloads resume seamlessly without re-downloading existing chunks.
- **Dual Mouse & Keyboard Navigation**:
  - Full mouse support: Click tabs, select rows, click action buttons (`[Pause]`, `[Logs]`, `[Del]`), and scroll with mouse wheel.
  - Intuitive single-key shortcuts (`j`/`k`, `Space`, `n`, `d`, `l`, `1-4`).
- **Real-Time Progress & Speed Metrics**: Displays real-time download speed, ETA countdown, total byte sizes, and progress percentage bars.
- **Live Curl Stderr Logs**: View exact `curl` commands and live output stream for each task.
- **Arch Desktop Integration**: Native desktop notifications via `notify-send` (`libnotify`) when downloads complete or error out.
- **State Persistence**: Download queue and progress persist automatically across application restarts in `~/.config/curler/state.json`.

---

## Installation on Arch Linux

### Building from Source

```bash
cd curler
go build -o curler .
sudo install -Dm755 curler /usr/bin/curler
```

## Usage

Start the TUI application:
```bash
curler
```

Or add a URL directly from command line:
```bash
curler -url https://releases.ubuntu.com/24.04/ubuntu-24.04-live-server-amd64.iso
```

---

## Navigation Reference

### Keyboard Shortcuts

| Key | Action |
| --- | --- |
| `n` | Open **New Download** prompt |
| `Space` / `p` | Toggle **Pause / Resume** on selected task |
| `Enter` / `l` | Open **Download Details & Live Curl Logs** |
| `d` / `Delete` | Open **Delete Confirmation** modal |
| `c` | **Clear completed** tasks from list |
| `1` - `4` | Switch tab filters (`1: All`, `2: Downloading`, `3: Completed`, `4: Paused`) |
| `Tab` / `Shift+Tab` | Cycle through tab filters |
| `j` / `k` or `↑`/`↓` | Navigate task list |
| `g` / `G` | Jump to top / bottom of task list |
| `?` | Show **Help** overlay |
| `Esc` | Close active modal |
| `q` / `Ctrl+C` | Exit application |

### Mouse Interactions

- **Left Click Header Tabs**: Switch filter view.
- **Left Click Task Row**: Select download task.
- **Left Click Buttons**: Trigger `[+ New]`, `[Pause/Resume]`, `[Logs]`, `[Del]`, `[Clear Done]`, `[Help]`.
- **Mouse Wheel**: Scroll task list or log viewport up/down.
