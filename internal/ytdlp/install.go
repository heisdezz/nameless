// Self-install and self-update of the yt-dlp binary into app-owned storage.
//
// When the system has no yt-dlp, the app downloads the official standalone
// release (github.com/yt-dlp/yt-dlp) into its own data directory and records
// the path in config.json (key "ytdlp_path"). Later launches prefer that
// binary, and Settings can check for newer releases or re-download.
package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Endpoints are package vars so tests can point them at a fake server.
var (
	// LatestReleaseURL resolves the newest yt-dlp release tag.
	LatestReleaseURL = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
	// LatestDownloadBase serves the standalone platform binaries.
	LatestDownloadBase = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"
)

// assetName returns the official standalone single-binary asset for this
// platform (Settings docs: yt-dlp provides exactly these).
func assetName() string {
	switch runtime.GOOS {
	case "windows":
		return "yt-dlp.exe"
	case "darwin":
		return "yt-dlp_macos"
	default:
		return "yt-dlp_linux"
	}
}

// LatestVersion queries GitHub for the newest yt-dlp release tag, e.g.
// "2026.10.26".
func LatestVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("ytdlp: latest release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// GitHub rejects requests without a UA.
	req.Header.Set("User-Agent", "nameless-desktop")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ytdlp: latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ytdlp: latest release: HTTP %d", resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("ytdlp: latest release: decode: %w", err)
	}
	tag := strings.TrimSpace(release.TagName)
	if tag == "" {
		return "", fmt.Errorf("ytdlp: latest release: empty tag_name")
	}
	return tag, nil
}

// IsNewerVersion reports whether latest sorts strictly after current
// (tags look like "2026.10.26"; painless numeric dot-comparison with a
// lexicographic fallback and equal handling for "v" prefixes).
func IsNewerVersion(current, latest string) bool {
	cur, lat := strings.TrimPrefix(current, "v"), strings.TrimPrefix(latest, "v")
	if cur == "" {
		return lat != ""
	}
	if lat == "" {
		return false
	}
	cs, ls := strings.Split(cur, "."), strings.Split(lat, ".")
	n := len(cs)
	if len(ls) > n {
		n = len(ls)
	}
	for i := 0; i < n; i++ {
		var ci, li int
		fmt.Sscanf(getPart(cs, i), "%d", &ci)
		fmt.Sscanf(getPart(ls, i), "%d", &li)
		if li != ci {
			return li > ci
		}
	}
	// All equal numeric parts; fall back to string comparison.
	return lat != cur && lat > cur
}

func getPart(parts []string, i int) string {
	if i < len(parts) {
		return strings.TrimSpace(parts[i])
	}
	return "0"
}

// DetectAt verifies that path is an executable yt-dlp by running its
// --version, mirroring Detect's 5s timeout and error shape.
func DetectAt(path string) (Binary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Binary{}, &NoBinaryError{Detail: "yt-dlp --version timed out after 5s"}
		}
		return Binary{}, &NoBinaryError{Detail: "yt-dlp --version failed: " + err.Error()}
	}
	ver := versionFromOutput(string(output))
	if ver == "" {
		return Binary{}, &NoBinaryError{Detail: "yt-dlp --version produced no version"}
	}
	return Binary{Path: path, Version: ver}, nil
}

// versionFromOutput extracts the version token (e.g. "2026.10.26") from the
// first line of --version output.
func versionFromOutput(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	out := fields[len(fields)-1]
	return out
}

// progressFunc receives cumulative byte counts during the download.
type progressFunc func(downloaded, total int64)

// InstallBinary downloads the latest standalone yt-dlp binary for this
// platform into destDir (creating it), atomically replacing any previous
// copy, marks the file executable, verifies "yt-dlp --version" runs, and
// returns the Binary. onProgress (if non-nil) receives cumulative progress.
func InstallBinary(ctx context.Context, destDir string, onProgress progressFunc) (Binary, error) {
	asset := assetName()
	dest := filepath.Join(destDir, asset)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return Binary{}, fmt.Errorf("ytdlp: install: create %s: %w", destDir, err)
	}

	url := LatestDownloadBase + asset
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Binary{}, fmt.Errorf("ytdlp: install: %w", err)
	}
	req.Header.Set("User-Agent", "nameless-desktop")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Binary{}, fmt.Errorf("ytdlp: install: download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Binary{}, fmt.Errorf("ytdlp: install: HTTP %d from %s", resp.StatusCode, url)
	}

	// Write to a .part temp next to the destination, then rename over the
	// previous binary so a crash can never leave a truncated executable.
	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return Binary{}, fmt.Errorf("ytdlp: install: create %s: %w", part, err)
	}
	total := resp.ContentLength
	n, err := io.Copy(io.MultiWriter(f, progressWriter{fn: onProgress, total: total}), resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(part)
		return Binary{}, fmt.Errorf("ytdlp: install: save %s: %w", dest, err)
	}
	// Standalone binaries must be executable to be usable.
	if err := os.Chmod(part, 0o755); err != nil {
		os.Remove(part)
		return Binary{}, fmt.Errorf("ytdlp: install: chmod %s: %w", dest, err)
	}
	if err := os.Rename(part, dest); err != nil {
		os.Remove(part)
		return Binary{}, fmt.Errorf("ytdlp: install: activate %s: %w", dest, err)
	}

	bin, verr := DetectAt(dest)
	if verr != nil {
		// A binary that does not run is worse than none: remove it.
		os.Remove(dest)
		return Binary{}, fmt.Errorf("ytdlp: install: downloaded binary failed %s check (%d bytes): %s", asset, n, verr.Error())
	}
	return bin, nil
}

// progressWriter forwards cumulative counters to the callback, safely
// tolerating an unknown total (Content-Length absent).
type progressWriter struct {
	fn    progressFunc
	total int64
	done  int64
}

func (w progressWriter) Write(p []byte) (int, error) {
	if w.fn != nil {
		w.done += int64(len(p))
		w.fn(w.done, w.total)
	}
	return len(p), nil
}
