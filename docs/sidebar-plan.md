# Architectural Plan: Sidebar + Main Content UI Redesign

**Project**: nameless (TikTok Bulk Downloader)  
**Target**: Desktop (Go + mygo/ebiten)  
**Author**: WS-E (Integrator)  
**Status**: Proposal / Ready for Implementation  

---

## 1. Overview & Motivation

The user requested: *"dont like the ui, i want a sidebar with main content"*.

The current prototype uses stacked, full-window screen transitions (`screenMain`, `screenProfile`, `screenSettings`) that replace the entire viewport. In a bulk downloader tool, desktop users benefit significantly from a persistent **sidebar layout**:
- **Constant orientation**: Recent profiles and quick navigation are always accessible without pressing a "← Back" button.
- **Parallel status visibility**: Global download state, worker count, or yt-dlp detection status can live continuously in the sidebar footer.
- **Natural hierarchy**: Left rail handles high-level navigation (New Download, History / Profiles, Settings), while the main viewport focuses on the task at hand (configuration, video batch review, download progress).

---

## 2. Layout Structure & Wireframe

```
+------------------------------------------------------------------------------------+
|  SIDEBAR (220-250px)            |  MAIN CONTENT AREA (Flex Fill)                   |
|---------------------------------+--------------------------------------------------|
|  [Logo] nameless                |  Header: Profile / Screen Title                  |
|  Bulk TikTok Downloader         |  Subtitle / Path / Context                       |
|---------------------------------+--------------------------------------------------|
|  NAVIGATION                     |                                                  |
|  [+] New Download               |  [ Dynamic View Area ]                           |
|                                 |                                                  |
|  RECENT PROFILES                |  - If "New Download":                            |
|  - @creator_one (42 vids)       |      Profile URL input, Destination folder,      |
|  - @artist_two  (128 vids)      |      Fetch button, quick hints                   |
|  - @dance_clips (15 vids)       |                                                  |
|                                 |  - If "Profile Selected":                        |
|  SETTINGS                       |      Stats bar (Done / In-flight / Failed),      |
|  [*] Settings & Engine          |      Batch actions (Download All, Pause, Retry), |
|                                 |      Virtualized list of videos with status,     |
|---------------------------------+      progress bars, and "Open folder" buttons.   |
|  SYSTEM / ENGINE STATUS         |                                                  |
|  yt-dlp: 2024.11.04 [Active]    |  - If "Settings":                                |
|  [Overall: 84/185 downloaded]   |      Theme tokens, Concurrency, Delays, Cookies  |
+---------------------------------+--------------------------------------------------+
```

---

## 3. Component & File Mapping (Ownership Preserved)

Per `docs/implementation-plan.md §5` exclusive ownership rules:

| File | Workstream | Role in Redesign |
| :--- | :---: | :--- |
| `ui_common.go` | **WS-E** | Root view container (`ui.Row` splitting sidebar and main area), toast drainage, theme apply wrapper. |
| `ui_sidebar.go` (or `ui_common.go`) | **WS-E** | Sidebar navigation rail: app branding, nav items, recent profiles list with active highlight, yt-dlp status footer. |
| `ui_main.go` | **WS-E** | "New Download" / Welcome surface: URL input, output directory picker, fetch action, spinner. |
| `ui_profile.go` | **WS-E** | Profile details surface: batch toolbar, overall progress, video list with virtualized items, per-row statuses, "Open folder" actions. |
| `ui_settings.go` | **WS-D** | Settings surface (fills main content area when Settings tab is active; no changes required to WS-D contracts). |
| `app.go` | **WS-E** | State extensions: active navigation tab (`navMain`, `navProfile`, `navSettings`), profile selection coordination. |

---

## 4. Key UX & Interaction Details

### 4.1. Navigation State
Replace discrete full-window swapping with a content selector:
```go
type navScreen int

const (
    navNewDownload navScreen = iota
    navProfileView
    navSettings
)
```

- Clicking **"[+] New Download"** switches main content to URL input.
- Clicking any profile in the **"Recent Profiles"** list sets `currentProfile` and switches main content to that profile's video grid/list.
- Fetching a new profile immediately creates/loads it and focuses its video list in the main view.
- Clicking **"Settings"** displays theme and engine options in the main view.

### 4.2. Responsive & Min-Size Handling (360×280 Min Window)
Desktop users may shrink the window down to the specified minimum (`360×280`):
- **Full desktop (≥ 600px)**: Sidebar displays full width (~200–220px) with profile names and status badges.
- **Compact mode (< 520px)**: Sidebar shrinks or collapses into a compact icon/shortcut rail (or toggleable drawer), preventing the main area from being squeezed out.
- Both sidebar and main view are independently scrollable (`ui.Scroll`) to ensure zero content clipping at 280px height.

### 4.3. Aesthetic & Theme Alignment
- Clean spacing: utilizes `c.Theme().Spacing` units.
- Rounded corners: container backgrounds and active nav buttons respect `c.Theme().Radius`.
- Subtle sidebar background (`t.Surface` or muted border `t.Border`) distinct from content area (`t.Background`).

---

## 5. Implementation Steps

1. **Step 1: State & Navigation Model (`app.go`)**
   - Align screen enum or add `activeNav` state without breaking existing tests.
   - Retain `openProfile(p)` and `enterSettings()` triggers.

2. **Step 2: Sidebar Component (`ui_common.go`)**
   - Implement `viewSidebar(c *ui.Context)`:
     - Header: App name + icon
     - Nav items: "New Download", "Settings"
     - Profiles list: Scrollable list of profiles with video counts and active selection indicator
     - Footer: Engine status & global progress indicator

3. **Step 3: Main Area Surface Integration (`ui_common.go`, `ui_main.go`, `ui_profile.go`)**
   - In `view(c *ui.Context)`: render `ui.Row` containing `viewSidebar(c)` and `viewContent(c)`.
   - Update `ui_main.go` and `ui_profile.go` to fit seamlessly as content panels (removing redundant "← Back" button since sidebar handles back navigation directly).

4. **Step 4: Testing & Verification**
   - Update `main_test.go` with `ui.NewTester` coverage:
     - Verify sidebar buttons navigate correctly between screens.
     - Verify selecting recent profiles from sidebar opens the profile view.
     - Verify layout does not panic or clip at min window size (360×280).
     - Run `go test -race ./...` and `go tool mygo vet .`.
