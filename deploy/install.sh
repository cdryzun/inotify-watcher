#!/bin/bash
#
# install.sh - Install inotify-hook service on TrueNAS SCALE
#
# Usage:
#   ./install.sh                    # Install with default paths
#   ./install.sh --uninstall        # Remove service
#

set -euo pipefail

# Configuration
BINARY_NAME="inotify-hook"
BINARY_SRC="./inotify-hook-linux-amd64"
BINARY_DST="/usr/local/bin/${BINARY_NAME}"
SERVICE_NAME="inotify-hook.service"
SERVICE_SRC="./deploy/${SERVICE_NAME}"
SERVICE_DST="/etc/systemd/system/${SERVICE_NAME}"
HOOK_SCRIPT_DIR="/root/scripts"
HOOK_SCRIPT="${HOOK_SCRIPT_DIR}/inotify-hook.sh"

log_info() {
    echo "[INFO] $*"
}

log_error() {
    echo "[ERROR] $*" >&2
}

log_success() {
    echo "[SUCCESS] $*"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

install_binary() {
    log_info "Installing binary..."

    if [[ ! -f "${BINARY_SRC}" ]]; then
        log_error "Binary not found: ${BINARY_SRC}"
        log_error "Please build first: GOOS=linux GOARCH=amd64 go build -o ${BINARY_SRC} ."
        exit 1
    fi

    cp "${BINARY_SRC}" "${BINARY_DST}"
    chmod +x "${BINARY_DST}"
    log_success "Binary installed: ${BINARY_DST}"
}

install_service() {
    log_info "Installing systemd service..."

    if [[ ! -f "${SERVICE_SRC}" ]]; then
        log_error "Service file not found: ${SERVICE_SRC}"
        exit 1
    fi

    cp "${SERVICE_SRC}" "${SERVICE_DST}"
    chmod 644 "${SERVICE_DST}"

    systemctl daemon-reload
    log_success "Service installed: ${SERVICE_DST}"
}

create_hook_script() {
    log_info "Checking hook script..."

    if [[ -f "${HOOK_SCRIPT}" ]]; then
        log_info "Hook script already exists: ${HOOK_SCRIPT}"
        return
    fi

    mkdir -p "${HOOK_SCRIPT_DIR}"

    cat > "${HOOK_SCRIPT}" << 'EOF'
#!/bin/bash
#
# inotify-hook.sh - Hook script for inotify-hook service
#
# Arguments:
#   $1 - Event type (CLOSE_WRITE, MOVED_TO, etc.)
#   $2 - Full file path
#   $3 - File name
#   $4 - Is directory (true/false)
#

set -euo pipefail

export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

EVENT_TYPE="${1:-}"
FILE_PATH="${2:-}"
FILE_NAME="${3:-}"
IS_DIR="${4:-false}"

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $*"
}

log_info "Event: ${EVENT_TYPE}, Path: ${FILE_PATH}, IsDir: ${IS_DIR}"

# Add your custom logic here
# Example: Trigger AList index refresh, send notifications, etc.

EOF

    chmod +x "${HOOK_SCRIPT}"
    log_success "Hook script created: ${HOOK_SCRIPT}"
    log_info "Please edit the hook script to add your custom logic"
}

enable_service() {
    log_info "Enabling service..."
    systemctl enable "${SERVICE_NAME}"
    log_success "Service enabled"
}

start_service() {
    log_info "Starting service..."
    systemctl start "${SERVICE_NAME}"
    log_success "Service started"

    echo ""
    log_info "Service status:"
    systemctl status "${SERVICE_NAME}" --no-pager || true
}

uninstall() {
    log_info "Uninstalling inotify-hook..."

    # Stop and disable service
    if systemctl is-active --quiet "${SERVICE_NAME}" 2>/dev/null; then
        systemctl stop "${SERVICE_NAME}"
        log_info "Service stopped"
    fi

    if systemctl is-enabled --quiet "${SERVICE_NAME}" 2>/dev/null; then
        systemctl disable "${SERVICE_NAME}"
        log_info "Service disabled"
    fi

    # Remove files
    [[ -f "${SERVICE_DST}" ]] && rm -f "${SERVICE_DST}" && log_info "Removed: ${SERVICE_DST}"
    [[ -f "${BINARY_DST}" ]] && rm -f "${BINARY_DST}" && log_info "Removed: ${BINARY_DST}"

    systemctl daemon-reload

    log_success "Uninstall complete"
    log_info "Hook script preserved: ${HOOK_SCRIPT}"
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    --uninstall    Remove service and binary
    --help         Show this help message

Examples:
    $0                 Install service
    $0 --uninstall     Remove service
EOF
}

main() {
    case "${1:-}" in
        --uninstall)
            check_root
            uninstall
            ;;
        --help|-h)
            show_usage
            ;;
        "")
            check_root
            install_binary
            install_service
            create_hook_script
            enable_service
            start_service

            echo ""
            log_success "Installation complete!"
            echo ""
            echo "Useful commands:"
            echo "  systemctl status ${SERVICE_NAME}    # Check status"
            echo "  systemctl restart ${SERVICE_NAME}   # Restart service"
            echo "  journalctl -u ${SERVICE_NAME} -f    # View logs"
            echo "  vim ${HOOK_SCRIPT}                  # Edit hook script"
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"
