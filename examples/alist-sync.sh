#!/bin/bash
#
# Example hook script for AList index synchronization
#
# This script refreshes the AList index when files are added or modified.
# Useful for ensuring AList reflects the latest file system state.
#
# Requirements:
#   - curl
#   - AList API endpoint
#   - AList admin token
#

set -euo pipefail

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

# Configuration
ALIST_API="http://localhost:5244"
ALIST_TOKEN="your-alist-admin-token"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# Only process file events (skip directories)
if [[ "$IS_DIR" == "true" ]]; then
    log "Skipping directory event: $FILE_PATH"
    exit 0
fi

# Extract the directory path
DIR_PATH=$(dirname "$FILE_PATH")

log "Refreshing AList index for: $DIR_PATH"

# Refresh AList index for the parent directory
# See: https://alist.nn.ci/api/
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$ALIST_API/api/admin/fs/refresh" \
    --header "Authorization: $ALIST_TOKEN" \
    --header "Content-Type: application/json" \
    --data "{
        \"path\": \"$DIR_PATH\"
    }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" == "200" ]]; then
    log "AList index refreshed successfully for: $DIR_PATH"
else
    log "Failed to refresh AList index: HTTP $HTTP_CODE"
    log "Response: $BODY"
    exit 1
fi
