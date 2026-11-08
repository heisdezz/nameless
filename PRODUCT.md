# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary users are desktop PC users who want to bulk-download all videos from a TikTok user profile. They are downloading for offline archiving, backup, or personal collection — not single-video grabs.

## Product Purpose

A desktop application that downloads every video from a given TikTok user profile using yt-dlp. Success means a user pastes a profile URL, picks a destination, and gets a complete, organized set of videos without manual per-video work.

## Positioning

Desktop-first bulk TikTok profile downloader driven by yt-dlp, with a clean, customizable, rounded-corner theme — unlike browser extensions or single-video tools.

## Operating Context

Runs on desktop PCs only (Windows/macOS/Linux via mygo/ebiten). Uses yt-dlp as the download engine. User supplies a profile URL and output location; the app orchestrates batch downloads and surfaces progress.

## Capabilities and Constraints

- Download all videos from a TikTok user profile (bulk, not single-video).
- Powered by yt-dlp.
- Desktop/PC only; no mobile or web target.
- Theme must be customizable, with clean spacing and rounded corners.
- yt-dlp availability and network access are external dependencies; their absence must be handled gracefully.

## Brand Commitments

- Name: "nameless".
- Visual constraint (user-stated, not expanded): customizable theme, clean spacing, rounded corners.

## Evidence on Hand

- `resources/icon.png` — app icon.
- `config.json` — yt-dlp-related settings (cookies, workers, delay, per-video timeout, ytdlp preference).
- No testimonials, benchmarks, or pricing exist; do not fabricate.

## Product Principles

- Bulk over single: optimize for downloading entire profiles, not one-off videos.
- Desktop-native feel: respect OS windowing and file-system conventions.
- Themeable from the start: spacing, corners, and colors are user-adjustable.
- Depend on yt-dlp, don't reinvent it: wrap, don't replace.
- Fail loudly and recoverably: missing yt-dlp or network errors are actionable, not silent.

## Accessibility & Inclusion

No product-specific accessibility standard established yet; follow platform conventions and keyboard operability for desktop apps.
