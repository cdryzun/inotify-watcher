#!/bin/bash
#
# Example hook script for sending notifications
#
# This script sends notifications via multiple channels (Slack, Email, etc.)
# when file system events occur.
#

set -euo pipefail

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

# Configuration
SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
EMAIL_RECIPIENT="admin@example.com"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

send_slack_notification() {
    local message="$1"

    curl -X POST "$SLACK_WEBHOOK" \
        --header "Content-Type: application/json" \
        --data "{
            \"text\": \"$message\",
            \"attachments\": [{
                \"color\": \"good\",
                \"fields\": [
                    {\"title\": \"Event Type\", \"value\": \"$EVENT_TYPE\", \"short\": true},
                    {\"title\": \"File Name\", \"value\": \"$FILE_NAME\", \"short\": true},
                    {\"title\": \"File Path\", \"value\": \"$FILE_PATH\", \"short\": false}
                ]
            }]
        }"

    log "Slack notification sent"
}

send_email_notification() {
    local subject="[Inotify Hook] File Event: $EVENT_TYPE"
    local body="
File System Event Detected

Event Type: $EVENT_TYPE
File Path: $FILE_PATH
File Name: $FILE_NAME
Is Directory: $IS_DIR
Timestamp: $(date)

--
TrueNAS Artifact Inotify Hook
"

    echo "$body" | mail -s "$subject" "$EMAIL_RECIPIENT"
    log "Email notification sent"
}

# Main logic
log "Processing event: $EVENT_TYPE - $FILE_PATH"

# Send notifications only for specific events
case "$EVENT_TYPE" in
    CLOSE_WRITE|MOVED_TO)
        # Notify on file uploads
        MESSAGE="📤 New file uploaded: $FILE_NAME"
        send_slack_notification "$MESSAGE"

        # Optionally send email for important files
        if [[ "$FILE_PATH" == *prod* ]]; then
            send_email_notification
        fi
        ;;

    DELETE)
        # Notify on file deletions
        MESSAGE="🗑️ File deleted: $FILE_NAME"
        send_slack_notification "$MESSAGE"
        ;;

    *)
        log "No notification for event type: $EVENT_TYPE"
        ;;
esac

log "Hook completed"
