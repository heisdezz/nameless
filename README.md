# nameless

**nameless** is a desktop bulk TikTok profile downloader powered by [`yt-dlp`](https://github.com/yt-dlp/yt-dlp) and built with Go and [`mygo`](https://github.com/egoist/mygo).

Unlike browser extensions or single-video downloaders, nameless is designed specifically for bulk archiving: paste a TikTok creator's profile URL, stream all their video metadata, and download entire profiles in parallel with live progress, pause/resume, and retry capabilities.

---

## Features

- **Bulk Profile Downloading**: Fetch and download every video from a TikTok profile without manual per-video links.
- **SQLite-Backed State & True Resume**: Download progress, metadata, and status (`pending`, `downloading`, `paused`, `done`, `failed`) are persisted to a local SQLite database (`data.db`). Paused or interrupted downloads preserve `.part` files and cleanly resume with `yt-dlp`'s `--continue` mechanism.
- **Concurrent Worker Pool**: Configurable parallel workers (1–16 concurrent downloads) with staggered launch delays to avoid hitting rate limits.
- **Resilient Queue Controls**: Download all, pause in-flight downloads, resume unfinished runs, or retry only failed videos with a single click.
- **Customizable Native UI**: Clean desktop interface with adjustable theme tokens (accent color hex, corner radius, spacing scale, dark mode toggle, and presets).
- **yt-dlp Detection & Power Settings**: Automatic binary detection with version reporting, duration-based timeouts, custom output directory, and optional cookie/session authentication for private or restricted accounts.
- **Cross-Platform & Pure Go**: Runs natively on Linux, macOS, and Windows with zero CGO requirements (`CGO_ENABLED=0`).

---

## Prerequisites

nameless uses `yt-dlp` as its underlying download engine. `yt-dlp` must be installed on your system and accessible in your `PATH`.

### Installing `yt-dlp`

Choose the package manager for your operating system:

#### Linux
- **Arch Linux**:
  ```bash
  sudo pacman -S yt-dlp
  ```
- **Debian / Ubuntu**:
  ```bash
  sudo apt update && sudo apt install yt-dlp
  ```
- **Fedora**:
  ```bash
  sudo dnf install yt-dlp
  ```
- **Pip / pipx (Universal)**:
  ```bash
  pipx install yt-dlp
  ```

#### macOS
- **Homebrew**:
  ```bash
  brew install yt-dlp
  ```
- **MacPorts**:
  ```bash
  sudo port install yt-dlp
  ```

#### Windows
- **winget**:
  ```powershell
  winget install yt-dlp.yt-dlp
  ```
- **Scoop**:
  ```powershell
  scoop install yt-dlp
  ```
- **Chocolatey**:
  ```powershell
  choco install yt-dlp
  ```

Verify your installation:
```bash
yt-dlp --version
```

---

## Building from Source

### Requirements
- **Go**: 1.27 or later
- **mygo**: Tooling for native Go desktop applications (`github.com/egoist/mygo`)

### Build Commands

To build using `mygo`:
```bash
go tool mygo build
```

Alternatively, build directly with the standard Go toolchain (pure Go SQLite, no C compiler required):
```bash
CGO_ENABLED=0 go build -o nameless
```

### Running Tests

Run the test suite:
```bash
CGO_ENABLED=0 go test ./...
```

Run static analysis with `mygo vet`:
```bash
go tool mygo vet .
```

---

## Usage Guide

1. **Launch the application**:
   ```bash
   ./nameless
   ```
2. **Fetch a Profile**:
   - Paste a profile URL into the **Profile URL** field (e.g., `https://www.tiktok.com/@username`).
   - Specify or confirm your **Save to** destination folder.
   - Click **Fetch profile**. nameless streams video entries via `yt-dlp` without buffering everything in memory.
3. **Download Videos**:
   - After fetching, you are navigated to the profile view showing the list of videos.
   - Click **Download all** to queue downloads across the worker pool.
   - Track live percentage and byte progress for each video.
   - Use **Pause** to suspend active downloads, **Resume** to continue, or **Retry failed** to re-queue any errored items.
4. **Settings**:
   - Click **Settings** from the main screen to customize theme colors, corner radiuses, and worker concurrency.
   - Configure session cookies (`Cookie`, `SessionID`, or path to an env file) if downloading from accounts requiring authentication.
   - Click **Detect** to refresh `yt-dlp` binary detection and version display.

---

## Configuration (`config.json`)

Settings are persisted in `config.json` in the application directory. You can adjust settings through the in-app Settings screen or by editing `config.json` directly:

```json
{
  "Cookie": "",
  "SessionID": "",
  "EnvFile": "",
  "DefaultDownloadLocation": "",
  "Workers": 3,
  "Delay": 1000000000,
  "PerVideoTimeout": 300000000000,
  "PreferYTDLP": true,
  "InstallYTDLP": false,
  "Theme": {
    "AccentHex": "#4f46e5",
    "Dark": false,
    "Radius": 12,
    "Spacing": 4
  }
}
```

### Key Settings

| Field | Description | Default |
|---|---|---|
| `Workers` | Number of concurrent download workers (1–16). | `3` |
| `Delay` | Delay between launching consecutive downloads (prevents aggressive rate limits). | `1s` |
| `PerVideoTimeout` | Maximum timeout before killing a stalled video download process. | `5m` |
| `DefaultDownloadLocation` | Default folder path where videos are saved. | System default |
| `Theme.AccentHex` | Primary accent color in hex (`#rrggbb`). | `#4f46e5` |
| `Theme.Dark` | Dark mode toggle (`true` / `false`). | `false` |
| `Theme.Radius` | UI element corner radius (0–24 DIP). | `12` |
| `Theme.Spacing` | Layout spacing unit (3 = Compact, 4 = Default, 5 = Roomy). | `4` |

---

## Data Storage

nameless stores profile history and video download progress in a SQLite database located at:
- **Linux**: `~/.config/nameless/data.db`
- **macOS**: `~/Library/Application Support/nameless/data.db`
- **Windows**: `%APPDATA%\nameless\data.db`

---

## License

This project is licensed under the MIT License. See `LICENSE` for details.
