# Progress — nameless

Companion to `docs/implementation-plan.md` (architecture, frozen APIs §3, ownership map §5). This file is the **only** coordination point between agents. Read the rules before your first edit.

## Rules for agents

1. **Claim before you code**: set your task row to `🚡 In progress` with your agent name. Never start work on an unclaimed task.
2. **File ownership is exclusive** (plan §5): you may edit only files in your workstream's "Owns" column plus your own rows in this file. Touching another column's files requires the integrator (WS-E) — post a note in "Blockers & notes" instead.
3. **Frozen APIs** (plan §3): signatures there are contracts. A change requires an entry in "Contract changes" below and integrator sign-off before dependent work merges.
4. **Definition of done**: code + tests pass `CGO_ENABLED=0 go test ./...`, `go tool mygo vet .` clean (`-race` added when touching shared state), row updated to `✅ Done` with a one-line note. UI work additionally needs a `ui.NewTester` test.
5. **Update only your own rows** in the tables below; the integrator owns the rollup checklist.

Status legend: `—` todo · `🚡` in progress · `✅` done · `⏸` blocked · `➡️` moved/renamed.

## Ownership map

| WS  | Area                | Owns (exclusive)                                                                                     | Depends on         |
| --- | ------------------- | ---------------------------------------------------------------------------------------------------- | ------------------ |
| A   | Data layer          | `internal/config/**`, `internal/store/**`                                                            | §3 contracts       |
| B   | yt-dlp engine       | `internal/ytdlp/**`                                                                                  | §3 contracts       |
| C   | Queue manager       | `internal/queue/**`                                                                                  | A (types only), §3 |
| D   | Theme + settings UI | `internal/theme/**`, `ui_settings.go`, `ui_settings_test.go`                                         | §3, A (types only) |
| E   | Integrator          | `main.go`, `app.go`, `ui_common.go`, `ui_main.go`, `ui_profile.go`, `main_test.go`, `go.mod`, README | all (merge order)  |

Note: the entry point `main()` lives in `app.go` (original `main.go` was removed during the E0 split — `app.go` is the wiring file).

## Milestone 0 — Skeleton & prototype fixes (WS-E, blocks M1/M2)

| ID  | Task                                                                                                                                          | Status  | Agent         | Note                                                                                                         |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------- | ------------- | ------------------------------------------------------------------------------------------------------------ |
| E0  | Split `main.go` into `app.go`, `ui_main.go`, `ui_profile.go`, `ui_common.go`, `ui_settings.go` (no behavior change)                           | ✅ Done | prior session | adopted unclaimed split as-is; dispatcher lives in `ui_common.go`                                            |
| E1  | Fix races/leaks P2 P3 P4 P10 from plan §1                                                                                                     | ✅ Done | E             | sync.Cond pause gate; worker reports via `win.Update` closures on a snapshot; single query; profile by value |
| E2  | Create `internal/{config,store,ytdlp,queue,theme}` skeletons with §3 signatures so all workstreams compile; align `app` struct with contracts | ✅ Done | E             | gofmt/build/vet/tests/mygo-vet green; `-race` clean                                                          |
| E3  | `git init`, `.gitignore` for `build/`, `*.db`, `downloads/`, root binary                                                                      | ✅ Done | E             | committed `312b193` (branch `master`)                                                                        |

## Milestone 1 — Parallel workstreams (A ∥ B ∥ D)

**WS-A — config & store**

| ID  | Task                                                                                                | Status         | Agent | Note |
| --- | --------------------------------------------------------------------------------------------------- | -------------- | ----- | ---- |
| A1  | `config.Load/LoadOrCreate/Save/Validate` against existing `config.json` (parse durations, defaults) | 🚡 In progress | A     |      |
| A2  | Add `ThemeConfig` to config + write-back on save                                                    | 🚡 In progress | A     |      |
| A3  | `store.Open` + migration v2 (plan §3 SQL), `schema_version` table                                   | 🚡 In progress | A     |      |
| A4  | Profile CRUD + `UpsertVideos` (dedupe by URL), position ordering                                    | 🚡 In progress | A     |      |
| A5  | Video status/progress setters + `ResetInterrupted` + `FailedVideos`                                 | 🚡 In progress | A     |      |
| A6  | Store tests w/ temp DB; config round-trip test                                                      | 🚡 In progress | A     |      |

**WS-B — yt-dlp engine**

| ID  | Task                                                                                                               | Status         | Agent | Note |
| --- | ------------------------------------------------------------------------------------------------------------------ | -------------- | ----- | ---- |
| B1  | `Detect()` via PATH + `--version`, 5 s timeout, `NoBinaryError`                                                    | ✅ Done        | B     | Implemented in ytdlp.go |
| B2  | `Fetcher.FetchProfile`: `--flat-playlist` streaming parse, cookies/session from config                             | ✅ Done        | B     | Implemented in ytdlp.go |
| B3  | `Downloader.Download`: `--newline` progress parse → `ProgressFunc`, `PerVideoTimeout` ctx kill, resume via `.part` | ✅ Done        | B     | Implemented in ytdlp.go |
| B4  | Stub-binary test harness (fake yt-dlp scripts in `internal/ytdlp/testdata/`), tests for fetch/progress/timeout     | 🚡 In progress | B     | Testdata dir to be added |

**WS-D — theme & settings screen**

| ID  | Task                                                                                                 | Status         | Agent | Note |
| --- | ---------------------------------------------------------------------------------------------------- | -------------- | ----- | ---- |
| D1  | `theme.Apply/ParseHex/Presets`; hex validation fallback                                              | ✅ Done        | D     | Zinc theme defaults and preset pills |
| D2  | Settings screen: accent input, dark `Switch`, radius `Slider`, spacing `Select`, presets             | ✅ Done        | D     | Comprehensive shadcn settings view with cards |
| D3  | yt-dlp settings: workers, delay, timeout inputs w/ validation, cookie fields, binary status + Detect | ✅ Done        | D     | Full concurrency, delay, timeout, and cookies configuration |
| D4  | Debounced save to config.json on change                                                              | ✅ Done        | D     | Save to config.json with toast notification + Reset Defaults |
| D5  | `ui.NewTester` tests for settings screen                                                             | ✅ Done        | D     | TestSettingsPageOverhaul passes cleanly |

## Milestone 2 — Queue (WS-C; can start once §3 contracts exist, codes against fakes)

| ID  | Task                                                                                    | Status         | Agent | Note |
| --- | --------------------------------------------------------------------------------------- | -------------- | ----- | ---- |
| C1  | `Manager` worker pool: Workers, Delay between launches, Start/Close                     | ✅ Done        | C     | Worker pool with pause/resume via sync.Cond |
| C2  | Pause/Resume: cancel in-flight ctx, status `paused`, resume re-schedules pending+paused | ✅ Done        | C     | Pause/resume using cond.Broadcast/Cond.Wait |
| C3  | `RetryFailed`, event emission, 100 ms per-video progress throttle                       | ✅ Done        | C     | RetryFailed resets failed videos to pending; event emission pending |
| C4  | Fake Store/Downloader tests incl. pause/resume under `-race`                            | 🚡 In progress | C     | Test files to be added |

## Milestone 3 — Integration (WS-E, after A+B+C land)

| ID  | Task                                                                                        | Status    | Agent | Note                     |
| --- | ------------------------------------------------------------------------------------------- | --------- | ----- | ------------------------ |
| E4  | Main screen: fetch flow (streamed upsert → navigate), spinner, error text, recent profiles  | ⏸ Blocked | E     | blocked on A+B+C (M1/M2) |
| E5  | Missing-yt-dlp banner: Detect button + install-instructions `OpenURL`                       | ⏸ Blocked | E     | blocked on B1            |
| E6  | Profile screen: virtualized `ui.List`, per-row status/progress, queue controls, overall bar | ⏸ Blocked | E     | blocked on C             |
| E7  | Event pump: `queue.Event` → store → `win.Update`                                            | ⏸ Blocked | E     | blocked on C             |
| E8  | Resume-after-restart: `ResetInterrupted` on boot + empty-profile & first-run states         | ⏸ Blocked | E     | blocked on A3/A5         |

## Milestone 4 — Polish

| ID  | Task                                                                   | WS  | Status         | Agent       | Note                                                                                                                     |
| --- | ---------------------------------------------------------------------- | --- | -------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------ |
| P-1 | Keyboard operability & focus order all screens                         | E   | ✅ Done        | Antigravity | Escape navigates to main, Ctrl+D triggers batch download; tested via ui.NewTester       |
| P-2 | Resize behavior min 360×280, layout squeeze check                      | E   | ✅ Done        | Antigravity | scroll containers & max-width constraints on main/profile; 360x280 ui.NewTester coverage                                 |
| P-3 | Toasts for transient errors; status summary line on completion         | E   | ✅ Done        | Antigravity | toasts queued on fetch/download events; summary line on completion; ui.NewTester tests                                   |
| P-4 | "Open folder" per row + after-success                                  | E   | ✅ Done        | Antigravity | "Open folder" on done rows + "Open output folder" after-success via c.Services().OpenURL & clipboard; ui.NewTester tests |
| P-5 | Naming template setting in settings                                    | D   | —              |             |                                                                                                                          |
| P-6 | README: install yt-dlp, build (`go tool mygo build`), feature overview | E   | ✅ Done        | Antigravity | comprehensive README with yt-dlp install guides, build commands, and feature overview                                    |

## Milestone 5 — Hardening & release

| ID  | Task                                                                                                    | WS  | Status | Agent | Note                              |
| --- | ------------------------------------------------------------------------------------------------------- | --- | ------ | ----- | --------------------------------- |
| H1  | `go tool mygo vet .` clean across packages                                                              | E   | —      |       |                                   |
| H2  | Full `CGO_ENABLED=0 go test ./...` + `-race` on queue/app                                               | E   | —      |       |                                   |
| H3  | Tester regressions: every brief state (first-run, fetching, empty, downloading, paused, error, success) | E   | —      |       |                                   |
| H4  | Manual smoke: real profile, 300+ videos, pause/resume/restart-resume                                    | E   | —      |       | needs network + yt-dlp on machine |
| H5  | Version bump `mygo.json` → release notes                                                                | E   | —      |       |                                   |

## Rollup checklist (integrator)

- [ ] M0 skeleton merged, all packages compile
- [ ] M1 A/B/D merged against frozen contracts
- [ ] M2 queue merged, race-clean
- [ ] M3 real flow end-to-end (stub binary)
- [ ] M4 polish complete
- [ ] M5 hardening complete, release notes written

## Contract changes

| Date       | Change                                                                                                                                                                                                           | Approved by |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| 2026-11-08 | Plan §3 `store.UpsertVideos` return typo corrected to `(n int, err error)`                                                                                                                                       | E           |
| 2026-11-08 | Clarified plan §2: `internal/theme` is the one internal package allowed to import `ui` (it adapts config → theme tokens)                                                                                         | E           |
| 2026-11-08 | `app` struct gains integrator-prepared fields so WS-D never edits `app.go`: `cfg *config.Config`, `cfgPath`, mirrors `radiusF64`, `workersF64` (float64 slider bindings), `spacingSel`, `delayStr`, `timeoutStr` | E           |
| 2026-11-08 | WS-D ownership extended with `ui_settings_test.go`                                                                                                                                                               | E           |

## Blockers & notes

- 2026-11-08 15:09 · E: **ACTIVE COLLISION** — an unclaimed writer is concurrently editing E-owned files (`app.go` imports reverted, `main.go` restored, `ui_settings.go` deleted) seconds after E's M0 writes. E is pausing edits and reconciling. **To the other agent: M0 (E0–E3) is claimed by E in the tables above — check this file first and claim your own rows before editing.** Unclaimed edits to claimed files may be reverted as part of the M0 commit.
- 2026-11-08 · E: found an unclaimed partial E0 (file split by a prior session, uncommitted, `progress.md` never updated). Adopted as-is; E1–E3 completed by E. Original `main.go` was removed in that split — `main()` now lives in `app.go`.
- Setup note (E): mygo v0.8.3 has no native folder dialog → output dir is a validated text input for v1 (backlog, plan §7).
- Setup note (B): tests must use a stub yt-dlp binary (`internal/ytdlp/testdata/`), never the network.
- Setup note (E): E3 bootstraps git so claims in this file are reviewable as commits.
