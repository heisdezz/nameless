# Design Brief — TikTok Bulk Downloader

Mode: Operate (task completion). Brand lives in precise details.

## Job & audience
Desktop users (casual + power) who paste a TikTok profile, fetch all its videos, and download them in parallel with pause/retry/resume. Casual gets a one-action flow; power gets yt-dlp flags, concurrency, naming, and theme controls via Settings.

## Outcome & proof
Primary task — paste URL → fetch profile → download all videos. Success = every video saved to chosen folder with visible progress, retryable failures, resumable runs. Real evidence: yt-dlp output, SQLite-backed download state, per-video status.

## Surfaces (3)
1. Main window — profile URL input, output folder picker, fetch + start, overall progress, status summary.
2. Profile/videos screen — list of fetched videos for a profile with per-item status, retry, open-in-folder.
3. Settings screen — theme customization (colors, spacing, corner radius), yt-dlp options, concurrency, compatibility mode.

## Selected direction
Operate mode — familiar desktop tool patterns. Clean spacing, rounded corners, themeable. SQLite persists download state for resume.

## Scope & boundaries
Production-ready screens, full flow, error/empty/loading states. Untouched: yt-dlp itself (wrap, don't replace). Anti-goals: no web target, no single-video-only mode, no invented marketing copy.

## States & ranges
First-run (no profile), fetching (loading), empty profile, downloading (partial progress), paused, error (missing yt-dlp/network), success (complete). Min 0 / typical 1–500 videos / max thousands.

## Interaction & layout
URL input → fetch → video list with status → download with parallel progress + pause/resume/retry → settings accessible but not in primary path. Responsive to window resize (min 360×280).

## Constraints
Go + mygo (ebiten) desktop, SQLite backend, yt-dlp dependency, desktop/PC only. Customizable theme (colors/spacing/corners). No CSS — native draw calls.

## Open decisions
Exact theme token defaults, settings persistence format, whether settings is separate window or in-app screen.
