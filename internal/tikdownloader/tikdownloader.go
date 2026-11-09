package tikdownloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultConnectionsPerDownload is the default number of parallel
// connections used for a single video file (not concurrent downloads).
const DefaultConnectionsPerDownload = 5

// VideoInfo holds metadata about a video from yt-dlp
type VideoInfo struct {
	ID        string
	URL       string
	Title     string
	Duration  string
	Thumbnail string
}

// DownloadResult represents the result of a download
type DownloadResult struct {
	Success bool
	Path    string
	Err     error
}

// Downloader interface for tikdownloader.io API
type Downloader interface {
	// DownloadVideo downloads a single video using tikdownloader.io API
	DownloadVideo(ctx context.Context, videoURL, outDir string) (*DownloadResult, error)
	// GetSupportedFormats returns supported format strings
	GetSupportedFormats() []string
}

// defaultDownloader implements Downloader using tikdownloader.io API
type defaultDownloader struct {
	apiEndpoint string
	apiCookies  string
	format      string
	// connections is the number of parallel Range connections used for a
	// single file download. It is distinct from how many videos may be
	// downloaded at once (config Workers / concurrent downloads).
	connections int
}

// New creates a new tikdownloader Downloader. connections <= 0 means the
// default (5); connections == 1 forces a single-stream download.
func New(apiEndpoint, apiCookies, format string, connections int) Downloader {
	if apiEndpoint == "" {
		apiEndpoint = "https://tikdownloader.io"
	}
	if format == "" {
		format = "mp4"
	}
	if connections <= 0 {
		connections = DefaultConnectionsPerDownload
	}
	return &defaultDownloader{
		apiEndpoint: apiEndpoint,
		apiCookies:  apiCookies,
		format:      format,
		connections: connections,
	}
}

// DownloadVideo downloads a single video using tikdownloader.io API
func (d *defaultDownloader) DownloadVideo(ctx context.Context, videoURL, outDir string) (*DownloadResult, error) {
	// Prepare API request. Use url.Values so URLs containing '&' or '?'
	// are not mangled (the previous fmt.Sprintf version truncated them).
	reqURL := d.apiEndpoint + "/api/ajaxSearch"

	form := url.Values{}
	form.Set("q", videoURL)
	form.Set("lang", "en")
	reqBody := form.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(reqBody))
	if err != nil {
		return &DownloadResult{Err: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://tikdownloader.io/")

	if d.apiCookies != "" {
		req.Header.Set("Cookie", d.apiCookies)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return &DownloadResult{Err: fmt.Errorf("API request failed: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &DownloadResult{Err: fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &DownloadResult{Err: fmt.Errorf("failed to read response: %w", err)}, nil
	}

	// Parse the HTML response to extract download URL
	downloadURL, err := extractDownloadURL(string(body))
	if err != nil {
		return &DownloadResult{Err: fmt.Errorf("failed to extract download URL: %w", err)}, nil
	}

	// Download the file
	outFilePath := filepath.Join(outDir, fmt.Sprintf("tiktok_%s.%s", generateSafeFilename(videoURL), d.format))

	if err := downloadFile(ctx, client, downloadURL, outFilePath, d.connections); err != nil {
		return &DownloadResult{Err: fmt.Errorf("download failed: %w", err)}, nil
	}

	return &DownloadResult{Success: true, Path: outFilePath}, nil
}

// downloadFile downloads url to outPath using up to connections parallel
// Range requests when the server supports them, falling back to a single
// stream otherwise. connections <= 1 always uses a single stream.
func downloadFile(ctx context.Context, client *http.Client, url, outPath string, connections int) error {
	if connections > 1 {
		if err := downloadFileParallel(ctx, client, url, outPath, connections); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		}
		// Fall through to single-stream on any parallel failure (e.g. the
		// CDN ignored Range). The partial file is truncated by Create below.
	}
	return downloadFileSingle(ctx, client, url, outPath)
}

func downloadFileSingle(ctx context.Context, client *http.Client, url, outPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

// downloadFileParallel probes for Range support with a 1-byte request and,
// on success, downloads the file in connections chunks concurrently,
// writing each chunk at its file offset with WriteAt.
func downloadFileParallel(ctx context.Context, client *http.Client, url, outPath string, connections int) error {
	total, err := probeRange(ctx, client, url)
	if err != nil || total <= 0 {
		if err == nil {
			err = fmt.Errorf("no Content-Range total")
		}
		return fmt.Errorf("range probe: %w", err)
	}
	if int64(connections) > total {
		connections = int(total)
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := out.Truncate(total); err != nil {
		return err
	}

	chunk := (total + int64(connections) - 1) / int64(connections)
	var wg sync.WaitGroup
	errCh := make(chan error, connections)
	for i := 0; i < connections; i++ {
		start := int64(i) * chunk
		if start >= total {
			break
		}
		end := start + chunk - 1
		if end >= total {
			end = total - 1
		}
		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()
			if err := downloadChunk(ctx, client, url, out, start, end); err != nil {
				errCh <- fmt.Errorf("chunk %d-%d: %w", start, end, err)
			}
		}(start, end)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func probeRange(ctx context.Context, client *http.Client, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("server returned %d, want 206", resp.StatusCode)
	}
	// Content-Range: bytes 0-0/<total>
	cr := resp.Header.Get("Content-Range")
	slash := strings.LastIndex(cr, "/")
	if slash < 0 {
		return 0, fmt.Errorf("missing Content-Range %q", cr)
	}
	return strconv.ParseInt(strings.TrimSpace(cr[slash+1:]), 10, 64)
}

func downloadChunk(ctx context.Context, client *http.Client, url string, out *os.File, start, end int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("chunk returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if int64(len(data)) != end-start+1 {
		return fmt.Errorf("chunk size %d, want %d", len(data), end-start+1)
	}
	_, err = out.WriteAt(data, start)
	return err
}

// Precompiled RE2 patterns. Go's regexp (RE2) has no \K and no lookahead,
// so the grep -P patterns from reference_scripts/dl.sh are rewritten with
// capture groups: group 1 is the URL. Using MustCompile per call with the
// old \K pattern panicked at runtime; these are valid RE2.
var (
	reMP4HD   = regexp.MustCompile(`href="([^"]+)"[^>]*>[^<]*<i[^>]*></i>[^<]*Download MP4 HD`)
	reSnapCDN = regexp.MustCompile(`href="([^"]*snapcdn[^"]*)"`)
	reV16     = regexp.MustCompile(`href="(https://v16[^"]*)"`)
)

// unescapeTikResponse mirrors dl.sh: the API returns JSON whose HTML is
// escaped (\r\n, \", \/). Undo that before parsing hrefs.
func unescapeTikResponse(s string) string {
	s = strings.ReplaceAll(s, `\r\n`, "")
	s = strings.ReplaceAll(s, `\n`, "")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\/`, "/")
	s = strings.ReplaceAll(s, `&amp;`, "&")
	return s
}

func cleanURL(u string) string {
	return strings.ReplaceAll(u, `&amp;`, "&")
}

// extractDownloadURL extracts the direct download URL from the API HTML response
func extractDownloadURL(html string) (string, error) {
	html = unescapeTikResponse(html)

	// Priority: href in the <a> tag whose text is "Download MP4 HD".
	if m := reMP4HD.FindStringSubmatch(html); len(m) > 1 && m[1] != "" {
		return cleanURL(m[1]), nil
	}

	// Fallback: first non-MP3 snapcdn link (iterate all, not just the first,
	// so an mp3 link first in the page does not hide a later video link).
	for _, m := range reSnapCDN.FindAllStringSubmatch(html, -1) {
		if len(m) > 1 && !strings.Contains(m[1], ".mp3") {
			return cleanURL(m[1]), nil
		}
	}

	// Last resort: direct CDN link.
	if m := reV16.FindStringSubmatch(html); len(m) > 1 && m[1] != "" {
		return cleanURL(m[1]), nil
	}

	return "", fmt.Errorf("could not extract download URL from API response")
}

// generateSafeFilename creates a safe filename from a URL
func generateSafeFilename(url string) string {
	// Extract the video ID or create a sanitized version
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Remove query parameters
		if idx := strings.Index(name, "?"); idx >= 0 {
			name = name[:idx]
		}
		// Sanitize
		name = strings.ReplaceAll(name, "/", "_")
		name = strings.ReplaceAll(name, "\\", "_")
		name = strings.ReplaceAll(name, ":", "_")
		return name
	}
	return "tiktok_video"
}

// GetSupportedFormats returns supported format strings
func (d *defaultDownloader) GetSupportedFormats() []string {
	return []string{
		"mp4",
		"hd",
		"720p",
		"1080p",
		"best",
	}
}

// Default creates a Downloader with default settings.
func Default() Downloader {
	return New("", "", "mp4", DefaultConnectionsPerDownload)
}

// WithOptions creates a Downloader with custom options.
func WithOptions(apiEndpoint, apiCookies, format string, connections int) Downloader {
	return New(apiEndpoint, apiCookies, format, connections)
}
