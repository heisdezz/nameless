#!/usr/bin/env bash
# Download all videos from a TikTok profile using dl.sh per video
# Supports resume via state.json

set -euo pipefail

PROFILE="${1:-}"

if [ -z "$PROFILE" ]; then
    echo "Usage: $0 <username|@username|profile-url>"
    echo "  Examples:"
    echo "    $0 username"
    echo "    $0 @username"
    echo "    $0 https://www.tiktok.com/@username"
    exit 1
fi

# Normalize input to a URL and a clean username for the output directory
if [[ "$PROFILE" == https://* ]]; then
    PROFILE_URL="$PROFILE"
    USERNAME=$(echo "$PROFILE" | grep -oP '(?<=@)[^/?]+' || echo "profile")
elif [[ "$PROFILE" == @* ]]; then
    USERNAME="${PROFILE#@}"
    PROFILE_URL="https://www.tiktok.com/@$USERNAME"
else
    USERNAME="$PROFILE"
    PROFILE_URL="https://www.tiktok.com/@$PROFILE"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DL_SCRIPT="$SCRIPT_DIR/dl.sh"

[ ! -f "$DL_SCRIPT" ] && echo "❌ dl.sh not found at: $DL_SCRIPT" && exit 1
command -v yt-dlp &>/dev/null || { echo "❌ yt-dlp is required"; exit 1; }
command -v python3 &>/dev/null || { echo "❌ python3 is required"; exit 1; }

# Load session cookies from .env for age-restricted content
YTDLP_ARGS=()
ENV_FILE="$SCRIPT_DIR/.env"
if [ -f "$ENV_FILE" ]; then
    COOKIE_STR=$(grep -v '^\s*#' "$ENV_FILE" | grep '=' | tr -d ' ' | tr '\n' ';' | sed 's/;$//')
    [ -n "$COOKIE_STR" ] && YTDLP_ARGS+=(--add-header "Cookie: $COOKIE_STR")
fi

mkdir -p "$USERNAME"

URLS_FILE="$USERNAME/.urls.txt"
STATE_FILE="$USERNAME/state.json"

# ── Phase 1: collect all URLs ────────────────────────────────────────────────
if [ ! -f "$URLS_FILE" ]; then
    echo "🔍 Fetching all video URLs from: $PROFILE_URL"
    # Write to a temp file so an interrupted run doesn't leave a partial .urls.txt
    TMP_URLS=$(mktemp)
    if yt-dlp --flat-playlist --print webpage_url "${YTDLP_ARGS[@]}" "$PROFILE_URL" > "$TMP_URLS"; then
        mv "$TMP_URLS" "$URLS_FILE"
    else
        rm -f "$TMP_URLS"
        echo "❌ Failed to fetch URL list"
        exit 1
    fi
    TOTAL=$(wc -l < "$URLS_FILE" | tr -d ' ')
    echo "📋 Found $TOTAL videos"
fi

TOTAL=$(wc -l < "$URLS_FILE" | tr -d ' ')

# Initialize state.json if missing (e.g. urls.txt existed from a prior run)
if [ ! -f "$STATE_FILE" ]; then
    python3 - <<EOF
import json
with open("$URLS_FILE") as f:
    urls = [l.strip() for l in f if l.strip()]
state = {"profile": "$USERNAME", "profile_url": "$PROFILE_URL",
         "total": len(urls), "done": [], "failed": []}
with open("$STATE_FILE", "w") as f:
    json.dump(state, f, indent=2)
EOF
fi

DONE_COUNT=$(python3 -c "import json; d=json.load(open('$STATE_FILE')); print(len(d['done']))")
echo "📋 $TOTAL total — $DONE_COUNT done, $((TOTAL - DONE_COUNT)) remaining — ./$USERNAME/"

# ── Phase 2: download ────────────────────────────────────────────────────────
COUNT=0
SKIPPED=0
FAILED=0

while IFS= read -r VIDEO_URL; do
    [ -z "$VIDEO_URL" ] && continue
    COUNT=$((COUNT + 1))

    VIDEO_ID=$(echo "$VIDEO_URL" | grep -oP '\d+$' || echo "")

    # Skip if already done
    if [ -n "$VIDEO_ID" ] && python3 -c "
import json, sys
d = json.load(open('$STATE_FILE'))
sys.exit(0 if '$VIDEO_ID' in d['done'] else 1)
" 2>/dev/null; then
        SKIPPED=$((SKIPPED + 1))
        echo "⏭️  [$COUNT/$TOTAL] already done: $VIDEO_ID"
        continue
    fi

    echo ""
    echo "── [$COUNT/$TOTAL] $VIDEO_URL"

    if (cd "$USERNAME" && bash "$DL_SCRIPT" "$VIDEO_URL" "$VIDEO_ID"); then
        [ -n "$VIDEO_ID" ] && python3 - <<EOF
import json
with open("$STATE_FILE") as f:
    state = json.load(f)
if "$VIDEO_ID" not in state["done"]:
    state["done"].append("$VIDEO_ID")
state["failed"] = [x for x in state["failed"] if x != "$VIDEO_ID"]
with open("$STATE_FILE", "w") as f:
    json.dump(state, f, indent=2)
EOF
        sleep 1
    else
        echo "⚠️  Failed: $VIDEO_URL"
        FAILED=$((FAILED + 1))
        [ -n "$VIDEO_ID" ] && python3 - <<EOF
import json
with open("$STATE_FILE") as f:
    state = json.load(f)
if "$VIDEO_ID" not in state["failed"]:
    state["failed"].append("$VIDEO_ID")
with open("$STATE_FILE", "w") as f:
    json.dump(state, f, indent=2)
EOF
    fi
done < "$URLS_FILE"

DOWNLOADED=$((COUNT - FAILED - SKIPPED))
echo ""
echo "✅ Done: $DOWNLOADED downloaded, $SKIPPED skipped, $FAILED failed — ./$USERNAME/"
echo "   Progress saved to ./$USERNAME/state.json"
