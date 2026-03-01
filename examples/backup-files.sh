#!/bin/bash
#
# Example hook script for automatic backup
#
# This script automatically backs up files when they are uploaded.
# Useful for creating redundant copies or versioned backups.
#

set -euo pipefail

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

# Configuration
BACKUP_DIR="/mnt/backup/artifacts"
MAX_BACKUPS=10
LOG_FILE="/var/log/inotify-backup.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Only backup CLOSE_WRITE events for files
if [[ "$EVENT_TYPE" != "CLOSE_WRITE" ]] || [[ "$IS_DIR" == "true" ]]; then
    exit 0
fi

log "Starting backup for: $FILE_PATH"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Generate timestamp for versioned backup
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')
BACKUP_NAME="${FILE_NAME}.${TIMESTAMP}.bak"
BACKUP_PATH="$BACKUP_DIR/$BACKUP_NAME"

# Copy file to backup location
if cp "$FILE_PATH" "$BACKUP_PATH"; then
    log "Backup created: $BACKUP_PATH"
else
    log "ERROR: Failed to create backup"
    exit 1
fi

# Cleanup old backups (keep only the last N backups)
log "Cleaning up old backups (keeping last $MAX_BACKUPS)..."

# Find and sort backups for this file
BACKUPS=($(find "$BACKUP_DIR" -name "${FILE_NAME}.*.bak" -type f | sort -r))
BACKUP_COUNT=${#BACKUPS[@]}

log "Found $BACKUP_COUNT backup(s) for $FILE_NAME"

if [[ $BACKUP_COUNT -gt $MAX_BACKUPS ]]; then
    # Delete old backups
    for ((i = MAX_BACKUPS; i < BACKUP_COUNT; i++)); do
        OLD_BACKUP="${BACKUPS[$i]}"
        log "Deleting old backup: $OLD_BACKUP"
        rm -f "$OLD_BACKUP"
    done
fi

# Calculate backup size
BACKUP_SIZE=$(du -sh "$BACKUP_PATH" | cut -f1)
log "Backup size: $BACKUP_SIZE"

log "Backup completed successfully"
