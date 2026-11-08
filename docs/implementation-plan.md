# Implementation Plan — nameless (TikTok Bulk Profile Downloader)

Source of truth: `PRODUCT.md`, `docs/design-brief.md`. Pinned stack: Go 1.27.3, mygo **v0.8.3** (`github.com/egoist/mygo`), `modernc.org/sqlite` (pure Go — keep `CGO_ENABLED=0` viable), yt-dlp as external engine.

Work is tracked in `progress.md`. This document defines architecture, package APIs (frozen contracts), and the workstream decomposition that lets multiple agents work simultaneously without file collisions.

---

## 1. Current state (audit of the prototype)

`main.go` is a working prototype with simulated downloads. Known issues to fix in M0:

| # | Issue | Location |
|---|-------|----------|
| P1 | Downloads are simulated — no yt-dlp invocation exists | `startDownload` |
| P2 | Busy-loop `for a.paused {}` pins a CPU core | `startDownload` |
| P3 | Data races: background goroutine calls `openProfile` (mutates `videos`, `screen`) and `a.fetching = false` without the mutex; `loadProfiles` runs on UI thread without lock | `fetchProfile`, `loadProfiles` |
| P4 | `loadProfiles` queries twice, first `rows`/`Close` leaked | `loadProfiles` |
| P5 | DB errors ignored everywhere (`db.Exec`, `rows.Scan`); `log.Fatal` on open failure kills app without UI | `initDB`, throughout |
| P6 | Video list not scrollable/virtualized (thousands of videos expected) | `screenProfile` |
| P7 | Settings screen: labels without controls; theme values never applied via `c.SetTheme` | `screenSettings` |
| P8 | No yt-dlp detection → brief's "missing yt-dlp" error state unimplemented | — |
| P9 | No pause coordination with downloads (pause only flips a bool the worker spins on) | `startDownload` |
| P10 | `openProfile` takes `*profile` pointing at a loop copy; pointer escapes after `append` may dangle logically | `openProfile`, `fetchProfile` |
| P11 | `config.json` exists but is never read by the app | — |

## 2. Target architecture

```
nameless/
├── main.go                  # wiring only: config load, store open, window, route to app
├── app.go                   # app struct, navigation, event pump (UI thread only)
├── ui_common.go             # shared widgets: banners, status badges, empty states
├── ui_main.go               # screen 1: URL input, output dir, fetch, recent profiles
├── ui_profile.go            # screen 2: video list, per-row status, queue controls
├── ui_settings.go           # screen 3: theme + yt-dlp settings        [WS-D owns]
├── internal/
│   ├── config/              # config.json load/save, defaults, validation   [WS-A]
│   ├── store/               # SQLite data layer + migrations                [WS-A]
│   ├── ytdlp/               # yt-dlp detection, profile fetch, download     [WS-B]
│   ├── queue/               # worker pool, pause/resume/retry, events       [WS-C]
│   └── theme/               # themeConfig → ui.Theme, presets, hex parse    [WS-D]
└── progress.md              # task tracking (each agent updates own rows only)
```

Dependency direction (no cycles): `queue → {store, ytdlp, config}`; `store → ytdlp` (for `VideoMeta`); `app → all internals`; internals never import package `main`. Single `ui` exception: `internal/theme` adapts `config.ThemeConfig` → `ui.Theme` tokens, so it may import `ui`.

### Threading model (per mygo rules)

- Only the UI thread mutates app state inside `view`. Background work (fetch, download queue) lives in internal packages and publishes results via `a.win.Update(fn)` — asynchronous, never blocks.
- The queue emits events on a channel; `app` runs one pump goroutine that selects on it and forwards into `win.Update`.
- Never store `ui.Element` or `*ui.Context` in state; store data + `ui.Handle` only. List rows get stable identity via `ListState` + `ItemKey`/`c.Key` so scroll position survives rebuilds.
- No busy-waiting anywhere; pause is queue state, not a spin loop.

## 3. Frozen package APIs

These signatures are the contract between workstreams. Changes require the integrator (WS-E) and an entry in `progress.md` → "Contract changes".

### `internal/config` (WS-A)

```go
type ThemeConfig struct {
    AccentHex string  // "#4f46e5"
    Dark      bool
    Radius    float32 // DIP, default 12
    Spacing   float32 // mygo spacing unit, default 4 (3 compact … 6 roomy)
}

type Config struct {
    Cookie, SessionID, EnvFile string        // as in existing config.json
    DefaultDownloadLocation    string
    Workers                    int           // default 3
    Delay                      time.Duration // parsed from "1s"
    PerVideoTimeout            time.Duration // parsed from "5m"
    PreferYTDLP, InstallYTDLP  bool
    Theme                      ThemeConfig
}

func Load(path string) (*Config, error)          // missing file → defaults, no error
func LoadOrCreate(path string) (*Config, error)  // writes defaults if absent
func (c *Config) Save(path string) error
func (c *Config) Validate() error                // Workers 1..16, durations > 0
```

### `internal/store` (WS-A)

```go
func Open(path string) (*DB, error)              // runs migrations, sets pragmas
func (db *DB) Close() error

type Profile struct{ ID int64; URL, Name string; FetchedAt time.Time; VideoCount int }
type Video struct {
    ID, ProfileID       int64
    URL, Title, Path    string
    Status              string // pending|downloading|paused|done|failed
    Pct                 float64
    Err                 string
    Downloaded, Total   int64  // bytes
    AddedAt, UpdatedAt  time.Time
}

// All methods return wrapped errors with context; no log.Fatal anywhere.
func (db *DB) CreateProfile(url, name string) (Profile, error)
func (db *DB) Profiles() ([]Profile, error)
func (db *DB) DeleteProfile(id int64) error
func (db *DB) UpsertVideos(profileID int64, vids []ytdlp.VideoMeta) (n int, err error)
func (db *DB) Videos(profileID int64) ([]Video, error)            // ordered: position then id
func (db *DB) SetVideoStatus(id int64, status string) error
func (db *DB) SetVideoProgress(id int64, pct float64, downloaded, total int64) error
func (db *DB) SetVideoFailed(id int64, errMsg, path string) error
func (db *DB) ResetInterrupted() error   // downloading/paused → pending at startup (resume)
func (db *DB) FailedVideos(profileID int64) ([]Video, error)
```

Schema (migration v2, additive; existing tables get `ALTER TABLE`):

```sql
-- v1 (exists): profiles(id,url,name), videos(id,profile_id,url,title,status,path)
-- v2:
ALTER TABLE profiles ADD COLUMN fetched_at INTEGER;
ALTER TABLE profiles ADD COLUMN video_count INTEGER DEFAULT 0;
ALTER TABLE videos  ADD COLUMN pct REAL DEFAULT 0;
ALTER TABLE videos  ADD COLUMN err TEXT DEFAULT '';
ALTER TABLE videos  ADD COLUMN downloaded_bytes INTEGER DEFAULT 0;
ALTER TABLE videos  ADD COLUMN total_bytes INTEGER DEFAULT 0;
ALTER TABLE videos  ADD COLUMN added_at INTEGER;
ALTER TABLE videos  ADD COLUMN updated_at INTEGER;
ALTER TABLE videos  ADD COLUMN position INTEGER;      -- playlist order
CREATE INDEX IF NOT EXISTS idx_videos_profile ON videos(profile_id);
CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL);
```

### `internal/ytdlp` (WS-B)

```go
type Binary struct{ Path, Version string }
func Detect() (Binary, error)          // PATH lookup + "yt-dlp --version" (5s timeout)

type VideoMeta struct{ ID, URL, Title string; Position int }

// FetchProfile streams line-delimited JSON from:
//   yt-dlp --flat-playlist --print "%(id)s\t%(url)s\t%(title)s" -j --newline <url>
// (exact flags finalized in B2; must handle 1000s of entries without buffering all in memory first,
//  honoring Config.Cookie / SessionID / EnvFile via --cookies / --username)
type Fetcher interface {
    FetchProfile(ctx context.Context, cfg *config.Config, profileURL string, onMeta func(VideoMeta)) error
}

type ProgressFunc func(p Progress)
type Progress struct{ Pct float64; Downloaded, Total int64 }

// Download runs one video to completion. outFile template example:
//   %(uploader)s/%(title).100B [%(id)s].%(ext)s
// Uses --newline to parse "[download]  42.3% of ..." lines; default yt-dlp resume
// (--continue, .part files) gives us resume for free after pause/kill.
type Downloader interface {
    Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ProgressFunc) (finalPath string, err error)
}

func NewFetcher(bin Binary) Fetcher
func NewDownloader(bin Binary) Downloader
```

- Both honor `PerVideoTimeout` via `context.WithTimeout` and kill the process on expiry.
- `NoBinaryError` sentinel type for the missing-yt-dlp case so UI can show the actionable state.
- Tests inject a **stub script** via a package-level `binaryPath` var or constructor param — never the network.

### `internal/queue` (WS-C)

```go
type Event struct {
    VideoID    int64
    Status     string // pending|downloading|paused|done|failed
    Pct        float64
    Err        string
    FinalPath  string
}

// Store and Downloader are small interfaces defined HERE (consumer side) so queue
// tests use fakes and never import ytdlp's concrete types.
type Store interface {
    Videos(profileID int64) ([]store.Video, error)
    SetVideoStatus(id int64, status string) error
    SetVideoProgress(id int64, pct float64, dl, total int64) error
    SetVideoFailed(id int64, msg, path string) error
}
type Downloader interface {
    Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ytdlp.ProgressFunc) (string, error)
}

func New(st Store, dl Downloader, cfg *config.Config, events chan<- Event) *Manager
func (m *Manager) Start(profileID int64, outDir string)   // schedules all pending videos, Workers at a time, Delay between launches
func (m *Manager) Pause()   // stop scheduling; kill running jobs (ctx cancel) → status "paused" (.part kept)
func (m *Manager) Resume()  // re-schedule pending+paused
func (m *Manager) RetryFailed() // failed → pending, reschedule if running
func (m *Manager) Close()   // cancel everything; idempotent
```

- One goroutine per worker pulls from an internal channel; `Delay` sleeps *before launch* so N workers stagger naturally.
- Pause semantics: cancel in-flight contexts → yt-dlp killed → `.part` preserved → status `paused`; Resume restarts with `--continue`, yt-dlp resumes the partial file.
- Events are emitted for every status/progress change; the UI pump throttles progress events to ≥100 ms per video.

### `internal/theme` (WS-D)

```go
func Apply(base *ui.Theme, cfg config.ThemeConfig) // mutates: Accent(+Hover/Pressed mix), Radius, Spacing, Dark base choice
func ParseHex(hex string) (ui.Color, error)        // accepts #rgb/#rrggbb, error → keep previous
func Presets() map[string]config.ThemeConfig       // "Indigo", "Forest", "Sunset", "Mono"
```

Applied in `app.view` top via `c.SetTheme(&t)` each build — cheap, matches mygo's "SetTheme makes the frame use t".

## 4. Feature implementation per screen

### Screen 1 — Main (`ui_main.go`, WS-E)

- Profile URL `ui.TextInput` with placeholder `https://www.tiktok.com/@username`; Fetch disabled while fetching (`ui.Spinner` inline); error text under input.
- Output folder: text input seeded from `config.DefaultDownloadLocation`; **Browse button deferred** (mygo v0.8.3 has no native folder dialog; avoid non-pure-Go deps — see Backlog).
- Recent profiles list → click opens profile screen. Empty state (first-run): short hint text "Paste a TikTok profile URL to begin".
- Missing yt-dlp: red banner "yt-dlp not found — install it, then press Detect" + button re-running `ytdlp.Detect()` + "Install instructions" via `c.Services().OpenURL("https://github.com/yt-dlp/yt-dlp#installation")`.
- Fetch flow: `FetchProfile` goroutine streams metas → `onMeta` batches → store upsert → on completion `win.Update` navigates to profile screen. Fetching state persists even if user navigates away (guard against double-fetch of same URL).

### Screen 2 — Profile (`ui_profile.go`, WS-E)

- Header: back, profile name, video count, `ui.Progress` overall bar (done/total), Download all / Pause / Resume / Retry failed buttons with correct enable/disable per state machine.
- Video list: `ui.List` + `ui.Scroll` (virtualized for thousands; stable row identity per mygo `ItemKey` guidance). Row: title (elided), status badge, per-video `ui.Progress` while downloading, error text in `t.Danger` for failed, "Open folder" per row (opens `filepath.Dir` via OS-specific `OpenURL`-style service if available; otherwise folder path with clipboard copy button).
- States from the brief: empty profile ("No videos found — the account may be private; try cookies in Settings"), downloading (live pct), paused (Resume affordance), failed rows (Retry failed), success (summary line "42/42 downloaded").
- Queue events pump in `app.go` maps `queue.Event` → store-backed video slice → `win.Update`.

### Screen 3 — Settings (`ui_settings.go`, WS-D)

- Theme: accent hex `TextInput` + live validation, dark `Switch`, radius `Slider` (0–24), spacing `Select` (Compact/Default/Roomy → 3/4/5), preset picker. Preview applied immediately via `SetTheme` next build; saved to `config.json` on change (debounced).
- yt-dlp: workers `Slider`/`Select` (1–16), delay and per-video timeout `TextInput` with duration validation, cookie/session fields, binary status line (version from `Detect`) + Detect button.
- Open decisions from the brief are hereby decided: **settings is an in-app screen** (not a separate window), **persistence is `config.json`** (theme tokens added to it), **theme token defaults** = accent `#4f46e5`, radius 12, spacing 4, dark off.

## 5. Workstreams & ownership (collision-free parallelism)

Each workstream **owns** the files in its column and must not edit files outside it. Cross-workstream needs go through the frozen APIs in §3 or via the integrator. `main.go`/`app.go` are edited only by WS-E. `progress.md`: each agent updates **only its own rows**.

| WS | Scope | Owns (exclusive) |
|----|-------|------------------|
| A | Data layer | `internal/config/**`, `internal/store/**` |
| B | yt-dlp engine | `internal/ytdlp/**` |
| C | Queue/manager | `internal/queue/**` |
| D | Theme + Settings UI | `internal/theme/**`, `ui_settings.go` |
| E | Integrator / main+profile UI | `main.go`, `app.go`, `ui_common.go`, `ui_main.go`, `ui_profile.go`, `main_test.go`, `go.mod` (deps), `progress.md` (rollup) |

Parallel-safe sequencing:

- **M0** (E alone, small): split `main.go` into `app.go`/`ui_*.go` files with zero behavior change; fix P2/P3/P4/P10 races and leaks; add `internal/` package skeletons with the §3 signatures returning `errors.New("not implemented")` so all workstreams compile from minute one.
- **M1** (A ∥ B ∥ D fully parallel; C starts after §3 is merged in M0): A implements config+store (+tests), B implements ytdlp fetch/download against a stub binary (+tests), D implements theme engine and the full settings screen against `config.Config` (compiles against A's package — if A hasn't landed, code against §3 signatures and the M0 skeleton).
- **M2** (C): queue manager against fake `Store`/`Downloader` (+tests incl. pause/resume races under `-race`).
- **M3** (E): wire real fetch → store → queue → UI; error banners; all brief states; resume-after-restart (`ResetInterrupted` on boot).
- **M4** (E ∥ D): keyboard operability, resize behavior (min 360×280), toasts, empty states, naming template setting, README.
- **M5** (E): hardening — `go tool mygo vet .`, full `CGO_ENABLED=0 go test ./...`, race checks on queue/app pump, `ui.NewTester` regression tests for all screens and state transitions, manual smoke test against a real profile.

## 6. Verification (every workstream, every task)

- `CGO_ENABLED=0 go test ./...` (pure-Go sqlite must keep working).
- `go tool mygo vet .` — catches stored build values and goroutine captures (mygo-specific).
- UI changes: tests with `ui.NewTester` exercising extra frames, focus, and element disappearance; never only manual.
- Shared state changes: `go test -race ./...`.
- No new dependencies without integrator approval (keep `CGO_ENABLED=0` viable; no CGO, no web views).

## 7. Backlog (explicitly out of scope for v1)

- Native OS folder picker (needs non-pure-Go dialog lib or future mygo support — revisit after v1).
- Managed yt-dlp install (`install_ytdlp` flag currently only surfaces instructions).
- Subtitle/metadata sidecar files, audio-only mode, single-video mode (anti-goal), multiple concurrent profiles.
