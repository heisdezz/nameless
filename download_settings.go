package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"nameless/internal/config"
	"nameless/internal/tikdownloader"
	"nameless/internal/ytdlp"
)

// This file wires the Settings fields that were previously persisted but
// never used: naming template, video quality, thumbnail/info/subtitle
// sidecars, cookie sources, delay, and per-video timeout.

// ytDLPBin resolves the yt-dlp executable: managed/probed binary first,
// then the configured custom path, then PATH lookup by name.
func (a *app) ytDLPBin(cfg *config.Config) string {
	if a.ytdlBin.Path != "" {
		return a.ytdlBin.Path
	}
	if cfg != nil && cfg.YTDLPPath != "" {
		return cfg.YTDLPPath
	}
	return "yt-dlp"
}

// effectiveTikFormat maps the Settings fields to a tikdownloader format.
// An explicitly non-default TikDownloaderFormat wins; otherwise a known
// VideoQuality ("720p", "1080p", "hd", "best", "mp4") is used.
// "audio_only" is unsupported by the tikdownloader.io MP4 endpoint, so it
// falls back to mp4 (the video is still downloaded).
func effectiveTikFormat(cfg *config.Config) string {
	if cfg == nil {
		return "mp4"
	}
	if f := cfg.TikDownloaderFormat; f != "" && f != "mp4" {
		return f
	}
	switch cfg.VideoQuality {
	case "720p", "1080p", "hd", "mp4":
		return cfg.VideoQuality
	case "best", "audio_only", "":
		return "mp4"
	default:
		if cfg.TikDownloaderFormat != "" {
			return cfg.TikDownloaderFormat
		}
		return "mp4"
	}
}

// renameWithTemplate renames a downloaded file according to the configured
// naming template, using yt-dlp entry metadata. Returns the final path;
// on any failure the original path is kept.
func (a *app) renameWithTemplate(cfg *config.Config, downloadedPath string, meta tikdownloader.VideoMeta) string {
	if cfg == nil || cfg.NamingTemplate == "" || downloadedPath == "" {
		return downloadedPath
	}
	ext := strings.TrimPrefix(filepath.Ext(downloadedPath), ".")
	if ext == "" {
		ext = "mp4"
	}
	name := tikdownloader.ApplyNamingTemplate(cfg.NamingTemplate, meta, ext)
	target := filepath.Join(filepath.Dir(downloadedPath), name)
	if target == downloadedPath {
		return downloadedPath
	}
	if _, err := os.Stat(target); err == nil {
		base := strings.TrimSuffix(name, "."+ext)
		target = filepath.Join(filepath.Dir(downloadedPath), base+"_"+meta.ID+"."+ext)
		if _, err := os.Stat(target); err == nil {
			return downloadedPath
		}
	}
	if err := os.Rename(downloadedPath, target); err != nil {
		return downloadedPath
	}
	return target
}

// downloadSidecars fetches thumbnail / info-JSON / subtitles next to an
// already-downloaded video using yt-dlp metadata mode (--skip-download).
// Best-effort: errors are returned but never fail the video download.
func (a *app) downloadSidecars(cfg *config.Config, videoURL, videoPath string) error {
	if cfg == nil || videoPath == "" {
		return nil
	}
	if !cfg.WriteThumbnail && !cfg.WriteInfoJSON && !cfg.WriteSubtitles {
		return nil
	}
	timeout := cfg.PerVideoTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ext := filepath.Ext(videoPath)
	base := strings.TrimSuffix(videoPath, ext)
	args := []string{"--skip-download", "--no-overwrites"}
	args = append(args, ytdlp.AuthArgs(cfg)...)
	if cfg.WriteThumbnail {
		args = append(args, "--write-thumbnail")
	}
	if cfg.WriteInfoJSON {
		args = append(args, "--write-info-json")
	}
	if cfg.WriteSubtitles {
		args = append(args, "--write-subtitles", "--write-auto-subs", "--sub-langs", "en.*")
	}
	args = append(args, "-o", base+".%(ext)s", videoURL)
	cmd := exec.CommandContext(ctx, a.ytDLPBin(cfg), args...)
	return cmd.Run()
}
