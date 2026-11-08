# Plan: Sleek Desktop Dashboard (Inspired by ref/dash.png)

## Overview & Reference Analysis

Following the user's reference image [`ref/dash.png`](file:///run/media/destiny/Ventoy/dr_projects/nameless/ref/dash.png), the UI transitions from a plain desktop form into a streaming-app-grade desktop dashboard:

1. **Left Nav Rail (Vertical Dock)**:
   - Floating rounded container with dark surface and subtle border.
   - App mark at top with glowing mint/teal accent.
   - Active navigation item rendered as an eye-catching rounded pill container (`#2dd4bf` accent) with high-contrast icon.
   - Inactive navigation icons (Home / New Download, Profiles, Settings) with hover highlights.

2. **Top Search & Destination Bar**:
   - Pill-shaped URL input with search icon placeholder: `🔍 https://www.tiktok.com/@username`.
   - Output destination folder badge/input with quick folder access.
   - "Fetch" action integrated into the pill layout.

3. **Hero Profile Card (Main Canvas)**:
   - High-contrast hero section with large profile name, video count, and download status.
   - Pill badge for active state (e.g. `● 84 Videos Ready`, `Downloading...`).
   - Primary mint-accent action button: `▶ Download All` / `⏸ Pause` / `▶ Resume`.
   - Quick toolbar actions: `Retry failed`, `Open folder`.

4. **Right Panel ("Queue / Recent Profiles")**:
   - Collapsible / persistent right column mirroring `ref/dash.png`'s "Up next" panel.
   - Lists recent profiles and queue progress with live status badges.

5. **Bottom Floating Download Dock (Mirroring Audio Player)**:
   - Full-width floating pill dock anchored to the bottom.
   - Left: Currently active video title & status indicator.
   - Center: Main transport controls (Pause/Resume, Retry) and overall batch progress bar (`X/Y downloaded`).
   - Right: Active worker indicators and direct "Open output folder" button.

---

## Technical Architecture & Implementation Steps

### Phase 1: Layout Primitives (`ui_common.go`)

- Create root layout composed of:
  - Outer `ui.Column` with dark background and margin/padding.
  - Middle `ui.Row`: Left Rail (`viewNavRail`), Central Content (`viewMain` or `viewProfile`), Right Panel (`viewQueuePanel`).
  - Bottom `viewBottomDock`: Floating status and batch download controls.

### Phase 2: Refined Surface Components (`ui_main.go`, `ui_profile.go`)

- `ui_main.go`: Top search pill, hero greeting card, quick instructions card.
- `ui_profile.go`: Hero profile card, video cards list with progress bars, status pills, and "Open folder" actions.

### Phase 3: Tests & Verification (`main_test.go`)

- `ui.NewTester` testing for:
  - Left rail navigation switches between New Download and Settings.
  - Right panel profile clicks load profile and videos.
  - Bottom dock controls (Pause/Resume/Open folder) respond accurately.
  - Minimum size (360×280) responsive behavior without panics or clipping.
  - `go test -race ./...` and `go tool mygo vet .`.
