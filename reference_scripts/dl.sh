#!/usr/bin/env bash
# TikTok downloader using tikdownloader.io API

set -euo pipefail

TIKTOK_URL="${1:-https://vt.tiktok.com/ZSxdcC9bj/}"
VIDEO_ID="${2:-}"  # optional: video ID passed by prof.sh for unique filenames
API_ENDPOINT="https://tikdownloader.io/api/ajaxSearch"

echo "📤 Submitting to tikdownloader.io API..."

# Submit request with proper headers to bypass Cloudflare
RESPONSE=$(curl -s -X POST "$API_ENDPOINT" \
    -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Referer: https://tikdownloader.io/" \
    -d "q=$TIKTOK_URL&lang=en")

# Check if response contains valid JSON status
if ! echo "$RESPONSE" | grep -q '"status"'; then
    echo "❌ API request failed"
    exit 1
fi

STATUS=$(echo "$RESPONSE" | grep -oP '"status"\s*:\s*"\K[^"]+' || echo "")
if [ "$STATUS" != "ok" ]; then
    echo "❌ API returned status: $STATUS"
    exit 1
fi

echo "✓ API response received"

# Extract title from HTML
TITLE=$(echo "$RESPONSE" | sed 's/\\r\\n/ /g' | grep -oP '<h3>[^<]*' | sed 's/<h3>//;s/&#[0-9]*;//g;s/&amp;/\&/g' | sed 's/[^a-zA-Z0-9 ]/_/g' | head -c 80)
[ -z "$TITLE" ] && TITLE="tiktok_download"
# Append video ID when provided to guarantee unique filenames across a profile
[ -n "$VIDEO_ID" ] && TITLE="${TITLE}_${VIDEO_ID}"

echo "📹 Video: $TITLE"

# Extract download URL - look for href values in the response
# Convert escaped newlines and quotes in JSON string
UNESCAPED=$(echo "$RESPONSE" | sed 's/\\r\\n//g' | sed 's/\\"/"/g' | sed 's/\\\//\//g')

# Priority: find href in the <a> tag whose text is "Download MP4 HD"
# The href comes before the label in the same tag, so use a lookahead to anchor on the label
DOWNLOAD_URL=$(echo "$UNESCAPED" | grep -oP 'href="\K[^"]+(?="[^>]*>[^<]*<i[^>]*></i>[^<]*Download MP4 HD)')

# Fallback: first non-MP3 snapcdn link (first link in dl-action)
if [ -z "$DOWNLOAD_URL" ]; then
    DOWNLOAD_URL=$(echo "$UNESCAPED" | grep -oP 'href="\K[^"]*dl\.snapcdn[^"]*' | grep -v '\.mp3' | head -1)
fi

# Last resort: direct CDN link
if [ -z "$DOWNLOAD_URL" ]; then
    DOWNLOAD_URL=$(echo "$UNESCAPED" | grep -oP 'href="\Khttps://v16[^"]*' | head -1)
fi

if [ -z "$DOWNLOAD_URL" ]; then
    echo "❌ Could not extract download URL"
    exit 1
fi

# Clean up URL
DOWNLOAD_URL=$(echo "$DOWNLOAD_URL" | sed 's/&amp;/\&/g')

echo "✅ Download URL extracted"
echo "📥 Downloading..."

# Use yt-dlp if available, otherwise curl
if command -v yt-dlp &> /dev/null 2>&1; then
    yt-dlp "$DOWNLOAD_URL" -o "${TITLE}.%(ext)s" && exit 0
fi

# Fallback to curl
curl -L --progress-bar -o "${TITLE}.mp4" "$DOWNLOAD_URL" && \
    echo "✅ Downloaded: ${TITLE}.mp4"
