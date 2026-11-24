package tikdownloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Supported downloader engine identifiers.
const (
	EngineAuto          = "auto"
	EngineTikDownloader = "tikdownloader"
	EngineTikWM         = "tikwm"
	EngineLoveTik       = "lovetik"
	EngineSSSTik        = "ssstik"
	EngineYTDLP         = "ytdlp"
)

const (
	defaultTikDownloaderURL = "https://tikdownloader.io"
	defaultTikWMURL         = "https://www.tikwm.com"
	defaultLoveTikURL       = "https://lovetik.com"
	defaultSSSTikURL        = "https://ssstik.io"
	defaultUserAgent        = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// EngineInfo describes a selectable TikTok download engine.
type EngineInfo struct {
	ID          string
	Label       string
	Description string
	DefaultURL  string
}

// AvailableEngines returns the ordered list of engines shown in Settings.
func AvailableEngines() []EngineInfo {
	return []EngineInfo{
		{
			ID:          EngineAuto,
			Label:       "Auto (Fallback)",
			Description: "Tries TikDownloader.io, TikWM, LoveTik, and SSSTik in order if one is rate-limited or down.",
		},
		{
			ID:          EngineTikDownloader,
			Label:       "TikDownloader.io",
			Description: "Uses tikdownloader.io API (supports custom endpoint and Cloudflare cookies).",
			DefaultURL:  defaultTikDownloaderURL,
		},
		{
			ID:          EngineTikWM,
			Label:       "TikWM",
			Description: "Uses tikwm.com JSON API (fast, no-watermark HD MP4 and MP3 audio).",
			DefaultURL:  defaultTikWMURL,
		},
		{
			ID:          EngineLoveTik,
			Label:       "LoveTik",
			Description: "Uses lovetik.com API (no-watermark MP4 and MP3 audio).",
			DefaultURL:  defaultLoveTikURL,
		},
		{
			ID:          EngineSSSTik,
			Label:       "SSSTik",
			Description: "Uses ssstik.io web endpoint (no-watermark MP4).",
			DefaultURL:  defaultSSSTikURL,
		},
		{
			ID:          EngineYTDLP,
			Label:       "yt-dlp",
			Description: "Downloads directly via the local yt-dlp binary using your TikTok cookies.",
		},
	}
}

// NormalizeEngine canonicalizes an engine name, defaulting empty or unknown
// values to EngineTikDownloader.
func NormalizeEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "", EngineTikDownloader, "tikdownloader.io":
		return EngineTikDownloader
	case EngineAuto:
		return EngineAuto
	case EngineTikWM, "tikwm.com":
		return EngineTikWM
	case EngineLoveTik, "lovetik.com":
		return EngineLoveTik
	case EngineSSSTik, "ssstik.io":
		return EngineSSSTik
	case EngineYTDLP, "yt-dlp":
		return EngineYTDLP
	default:
		return EngineTikDownloader
	}
}

func isAudioFormat(format string) bool {
	f := strings.ToLower(strings.TrimSpace(format))
	return f == "audio_only" || f == "mp3" || f == "audio"
}

func endpointForEngine(engine, configured string) string {
	configured = strings.TrimRight(strings.TrimSpace(configured), "/")
	switch engine {
	case EngineTikDownloader:
		if configured == "" {
			return defaultTikDownloaderURL
		}
		return configured
	case EngineTikWM:
		if configured == "" || isKnownDefaultEndpoint(configured) {
			return defaultTikWMURL
		}
		return configured
	case EngineLoveTik:
		if configured == "" || isKnownDefaultEndpoint(configured) {
			return defaultLoveTikURL
		}
		return configured
	case EngineSSSTik:
		if configured == "" || isKnownDefaultEndpoint(configured) {
			return defaultSSSTikURL
		}
		return configured
	default:
		return defaultTikDownloaderURL
	}
}

func isKnownDefaultEndpoint(u string) bool {
	switch strings.TrimRight(u, "/") {
	case defaultTikDownloaderURL, defaultTikWMURL, defaultLoveTikURL, defaultSSSTikURL:
		return true
	default:
		return false
	}
}

func httpStatusError(engine string, status int, body []byte) error {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 160 {
		snippet = snippet[:160] + "..."
	}
	switch {
	case status == http.StatusTooManyRequests:
		return fmt.Errorf("[%s] rate-limited (HTTP 429): %s", engine, snippet)
	case status == http.StatusForbidden:
		return fmt.Errorf("[%s] access blocked (HTTP 403): %s", engine, snippet)
	case status >= 500:
		return fmt.Errorf("[%s] service unavailable (HTTP %d): %s", engine, status, snippet)
	default:
		return fmt.Errorf("[%s] API returned status %d: %s", engine, status, snippet)
	}
}

// resolveTikDownloader queries tikdownloader.io /api/ajaxSearch.
func resolveTikDownloader(ctx context.Context, client *http.Client, baseURL, cookies, videoURL, format string) (string, string, error) {
	reqURL := strings.TrimRight(baseURL, "/") + "/api/ajaxSearch"

	form := url.Values{}
	form.Set("q", videoURL)
	form.Set("lang", "en")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("[%s] create request: %w", EngineTikDownloader, err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", strings.TrimRight(baseURL, "/")+"/")
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("[%s] request failed: %w", EngineTikDownloader, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("[%s] read response: %w", EngineTikDownloader, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", httpStatusError(EngineTikDownloader, resp.StatusCode, body)
	}

	dlURL, err := extractDownloadURL(string(body))
	if err != nil {
		return "", "", fmt.Errorf("[%s] %w", EngineTikDownloader, err)
	}
	return dlURL, "mp4", nil
}

// resolveTikWM queries tikwm.com JSON API.
func resolveTikWM(ctx context.Context, client *http.Client, baseURL, videoURL, format string) (string, string, error) {
	base := strings.TrimRight(baseURL, "/")
	reqURL := base + "/api/"

	form := url.Values{}
	form.Set("url", videoURL)
	form.Set("hd", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("[%s] create request: %w", EngineTikWM, err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", base+"/")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("[%s] request failed: %w", EngineTikWM, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("[%s] read response: %w", EngineTikWM, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", httpStatusError(EngineTikWM, resp.StatusCode, body)
	}

	var parsed struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Play   string `json:"play"`
			HDPlay string `json:"hdplay"`
			WMPlay string `json:"wmplay"`
			Music  string `json:"music"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", fmt.Errorf("[%s] decode JSON: %w", EngineTikWM, err)
	}
	if parsed.Code != 0 {
		msg := parsed.Msg
		if msg == "" {
			msg = "unknown error"
		}
		return "", "", fmt.Errorf("[%s] API error (code %d): %s", EngineTikWM, parsed.Code, msg)
	}

	var chosen, ext string
	if isAudioFormat(format) && parsed.Data.Music != "" {
		chosen = parsed.Data.Music
		ext = "mp3"
	} else {
		ext = "mp4"
		switch {
		case strings.EqualFold(format, "sd") && parsed.Data.Play != "":
			chosen = parsed.Data.Play
		case parsed.Data.HDPlay != "":
			chosen = parsed.Data.HDPlay
		case parsed.Data.Play != "":
			chosen = parsed.Data.Play
		case parsed.Data.WMPlay != "":
			chosen = parsed.Data.WMPlay
		}
	}
	if chosen == "" {
		return "", "", fmt.Errorf("[%s] no download URL in API response", EngineTikWM)
	}
	if strings.HasPrefix(chosen, "/") {
		chosen = base + chosen
	}
	return chosen, ext, nil
}

// resolveLoveTik queries lovetik.com /api/ajax/search.
func resolveLoveTik(ctx context.Context, client *http.Client, baseURL, videoURL, format string) (string, string, error) {
	base := strings.TrimRight(baseURL, "/")
	reqURL := base + "/api/ajax/search"

	form := url.Values{}
	form.Set("query", videoURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("[%s] create request: %w", EngineLoveTik, err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", base+"/")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("[%s] request failed: %w", EngineLoveTik, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("[%s] read response: %w", EngineLoveTik, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", httpStatusError(EngineLoveTik, resp.StatusCode, body)
	}

	var parsed struct {
		Status string `json:"status"`
		Mess   string `json:"mess"`
		Links  []struct {
			A string `json:"a"`
			T string `json:"t"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", fmt.Errorf("[%s] decode JSON: %w", EngineLoveTik, err)
	}
	if !strings.EqualFold(parsed.Status, "ok") {
		msg := parsed.Mess
		if msg == "" {
			msg = parsed.Status
		}
		return "", "", fmt.Errorf("[%s] API error: %s", EngineLoveTik, msg)
	}

	wantAudio := isAudioFormat(format)
	var fallback string
	for _, l := range parsed.Links {
		if l.A == "" {
			continue
		}
		label := strings.ToLower(l.T)
		isAudio := strings.Contains(label, "mp3") || strings.Contains(label, "audio")
		if wantAudio {
			if isAudio {
				return cleanURL(l.A), "mp3", nil
			}
			continue
		}
		if isAudio {
			continue
		}
		if strings.Contains(label, "1080") || strings.Contains(label, "hd") || strings.Contains(label, "no watermark") {
			return cleanURL(l.A), "mp4", nil
		}
		if fallback == "" {
			fallback = l.A
		}
	}
	if fallback != "" {
		return cleanURL(fallback), "mp4", nil
	}
	return "", "", fmt.Errorf("[%s] no matching download link in API response", EngineLoveTik)
}

var (
	reSSSTikToken   = regexp.MustCompile(`s_tt\s*=\s*['"]([^'"]+)['"]`)
	reSSSTikInputTT = regexp.MustCompile(`name="tt"\s+value="([^"]+)"`)
	reSSSTikNoWM    = regexp.MustCompile(`href="([^"]+)"[^>]*class="[^"]*without_watermark[^"]*"`)
	reSSSTikNoWMAlt = regexp.MustCompile(`class="[^"]*without_watermark[^"]*"[^>]*href="([^"]+)"`)
	reSSSTikTikCDN  = regexp.MustCompile(`href="(https://[^"]*tikcdn[^"]+)"`)
)

// resolveSSSTik fetches the tt token from ssstik.io and submits the video URL.
func resolveSSSTik(ctx context.Context, client *http.Client, baseURL, videoURL, format string) (string, string, error) {
	base := strings.TrimRight(baseURL, "/")

	// Step 1: fetch the page to extract the s_tt token.
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/en", nil)
	if err != nil {
		return "", "", fmt.Errorf("[%s] create token request: %w", EngineSSSTik, err)
	}
	getReq.Header.Set("User-Agent", defaultUserAgent)

	getResp, err := client.Do(getReq)
	if err != nil {
		return "", "", fmt.Errorf("[%s] token request failed: %w", EngineSSSTik, err)
	}
	pageBody, err := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if err != nil {
		return "", "", fmt.Errorf("[%s] read token page: %w", EngineSSSTik, err)
	}
	if getResp.StatusCode != http.StatusOK {
		return "", "", httpStatusError(EngineSSSTik, getResp.StatusCode, pageBody)
	}

	var token string
	if m := reSSSTikToken.FindSubmatch(pageBody); len(m) > 1 {
		token = string(m[1])
	} else if m := reSSSTikInputTT.FindSubmatch(pageBody); len(m) > 1 {
		token = string(m[1])
	}
	if token == "" {
		return "", "", fmt.Errorf("[%s] could not extract session token", EngineSSSTik)
	}

	// Step 2: POST to /abc?url=dl.
	form := url.Values{}
	form.Set("id", videoURL)
	form.Set("locale", "en")
	form.Set("tt", token)

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/abc?url=dl", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", fmt.Errorf("[%s] create download request: %w", EngineSSSTik, err)
	}
	postReq.Header.Set("User-Agent", defaultUserAgent)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("HX-Request", "true")
	postReq.Header.Set("Referer", base+"/en")
	postReq.Header.Set("Origin", base)

	postResp, err := client.Do(postReq)
	if err != nil {
		return "", "", fmt.Errorf("[%s] download request failed: %w", EngineSSSTik, err)
	}
	defer postResp.Body.Close()

	respBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		return "", "", fmt.Errorf("[%s] read download response: %w", EngineSSSTik, err)
	}
	if postResp.StatusCode != http.StatusOK {
		return "", "", httpStatusError(EngineSSSTik, postResp.StatusCode, respBody)
	}

	html := string(respBody)
	for _, re := range []*regexp.Regexp{reSSSTikNoWM, reSSSTikNoWMAlt, reSSSTikTikCDN} {
		if m := re.FindStringSubmatch(html); len(m) > 1 && m[1] != "" {
			return cleanURL(m[1]), "mp4", nil
		}
	}
	return "", "", fmt.Errorf("[%s] could not extract download URL from response", EngineSSSTik)
}
