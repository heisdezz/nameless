# Architectural Plan: Dashboard Home Overview Stats & Settings Consolidation

**Project**: nameless (TikTok Bulk Downloader)  
**Location**: `docs/dashboard-stats-plan.md`  
**Target**: Desktop (Go + mygo/ebiten)  
**Author**: WS-E (Integrator)  
**Status**: Proposal / Ready for Implementation  

---

## 1. User Direction & Objective

The user requested:
> *"for the dashboard (home section), want overview stats, along with the button, download settings and every other thing should be in settings page"*

### Key Changes:
1. **Dashboard (Home Section — `ui_main.go`)**:
   - Strip out configuration inputs like "Save to" / output directory from the main home view.
   - Introduce **Overview Stats Cards** providing at-a-glance metrics:
     - **Total Videos Downloaded** (across all profiles)
     - **Profiles Archived** (count of saved profiles)
     - **Active Queue Status** (Idle / Downloading / Paused)
     - **Engine Health** (yt-dlp version / status)
   - Keep a clean, focused **URL Input + Action Button**:
     - Modern pill-shaped search input for `@username` or TikTok profile URL.
     - Prominent **"Fetch Profile"** primary CTA button.
     - Welcome / Quick-Start Hero Card without configuration clutter.

2. **Settings Screen (`ui_settings.go`)**:
   - Consolidate **all** download and app settings in one place:
     - **Storage & Output**: Output directory ("Save to" path) with current default (`./downloads`).
     - **Engine Configuration**: Concurrency workers slider (1–16), per-video delay, per-video timeout.
     - **Authentication & Cookies**: Session ID, Cookie file, .env file.
     - **Theme & Appearance**: Accent color hex, Dark mode switch, Corner radius slider, Spacing select, Presets.

---

## 2. Layout Wireframe

### 2.1 Home / Dashboard (`ui_main.go`)
```
+---------------------------------------------------------------------------------------------+
|  [ 🔍 https://www.tiktok.com/@username                                    [Fetch Profile] ] |
+---------------------------------------------------------------------------------------------+
|                                                                                             |
|  OVERVIEW STATS                                                                             |
|  +--------------------+  +--------------------+  +--------------------+  +----------------+ |
|  | 📥 142 Videos      |  | 📁 8 Profiles      |  | ⚡ 3 Workers       |  | ≋ yt-dlp       | |
|  | Total Downloaded   |  | Archived           |  | Queue: Idle        |  | Version Active | |
|  +--------------------+  +--------------------+  +--------------------+  +----------------+ |
|                                                                                             |
|  HERO QUICK-START CARD                                                                      |
|  +----------------------------------------------------------------------------------------+ |
|  | ● BULK TIKTOK ARCHIVE                                                                  | |
|  | Start a New Profile Download                                                           | |
|  | Paste a profile link above and click "Fetch Profile" to batch-download all videos.     | |
|  | Download location, concurrency, and cookies are managed in Settings.                   | |
|  +----------------------------------------------------------------------------------------+ |
|                                                                                             |
+---------------------------------------------------------------------------------------------+
```

### 2.2 Settings Screen (`ui_settings.go`)
```
+---------------------------------------------------------------------------------------------+
|  Settings                                                                                   |
|---------------------------------------------------------------------------------------------|
|  DOWNLOAD SETTINGS                                                                          |
|  Default Download Location: [ ./downloads                                ]                  |
|  Concurrency Workers:       [=====|===========] (3 workers)                                 |
|  Delay between requests:    [ 1s              ]                                             |
|  Per-video Timeout:         [ 5m              ]                                             |
|                                                                                             |
|  TIKTOK AUTHENTICATION (FOR PRIVATE/RESTRICTED PROFILES)                                    |
|  Cookie File:               [                                            ]                  |
|  Session ID:                [                                            ]                  |
|                                                                                             |
|  THEME & APPEARANCE                                                                         |
|  Accent Hex: [ #2dd4bf   ]   Dark Mode: [ON]   Radius: [==|====] (16px)  Spacing: [Default] |
+---------------------------------------------------------------------------------------------+
```

---

## 3. Data & State Additions

To power the **Overview Stats** cleanly on the Home screen without running heavy queries on every frame:
- Helper function `(a *app) overviewStats() (profilesCount int, downloadedCount int, totalCount int)`:
  - Queries SQLite or summarizes in-memory state:
    - `profilesCount = len(a.profiles)`
    - `downloadedCount` from `SELECT COUNT(*) FROM videos WHERE status='done'`
    - `totalCount` from `SELECT COUNT(*) FROM videos`
  - Cached/refreshed on profile changes or download completions.

---

## 4. Implementation Steps

1. **Step 1: Metric Helper in `app.go`**:
   - Add `overviewStats()` returning total downloaded videos, total profiles, and active status.
2. **Step 2: Redesign `ui_main.go`**:
   - Remove output directory input from the home view.
   - Render 4 stat cards in a responsive row:
     1. Downloaded Videos count.
     2. Profiles count.
     3. Worker / Queue status.
     4. yt-dlp version / engine status.
   - Top search-style URL input with "Fetch profile" CTA.
3. **Step 3: Update `ui_settings.go`**:
   - Add "Download Settings" section containing:
     - Output folder text input (`&a.outDir` or `&a.cfg.DefaultDownloadLocation`).
     - Workers slider, Delay text input, Timeout text input.
     - Cookie & session inputs.
4. **Step 4: Update Tests in `main_test.go`**:
   - Verify overview stats cards render on `screenMain`.
   - Verify settings page includes download settings and output directory.
   - Run `go test -race ./...` and `go tool mygo vet .`.
