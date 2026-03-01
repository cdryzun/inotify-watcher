#!/bin/bash
#
# Example hook script for triggering CI/CD pipelines
#
# This script demonstrates how to trigger a CI/CD pipeline
# when a new artifact is uploaded.
#
# Usage:
#   Configure this script as your hook:
#   inotify-hook watch /data/artifacts --hook=/path/to/this/script.sh
#

set -euo pipefail

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# Only process CLOSE_WRITE events for files (not directories)
if [[ "$EVENT_TYPE" != "CLOSE_WRITE" ]] || [[ "$IS_DIR" == "true" ]]; then
    exit 0
fi

log "Processing artifact: $FILE_PATH"

# Example: Trigger Jenkins job
if [[ "$FILE_PATH" == *.tar.gz ]]; then
    log "Triggering Jenkins job for artifact: $FILE_NAME"

    curl -X POST "https://jenkins.example.com/job/deploy/buildWithParameters" \
        --user "jenkins-user:api-token" \
        --data "ARTIFACT_PATH=$FILE_PATH" \
        --data "ARTIFACT_NAME=$FILE_NAME" \
        --data "TOKEN=your-token"

    log "Jenkins job triggered successfully"
fi

# Example: Trigger GitLab CI pipeline
if [[ "$FILE_PATH" == *-prod-*.tar.gz ]]; then
    log "Triggering production deployment pipeline"

    curl -X POST "https://gitlab.example.com/api/v4/projects/123/trigger/pipeline" \
        --form "token=your-trigger-token" \
        --form "ref=main" \
        --form "variables[ARTIFACT_PATH]=$FILE_PATH" \
        --form "variables[DEPLOY_ENV]=production"

    log "GitLab CI pipeline triggered"
fi

log "Hook completed for: $FILE_PATH"
