#!/bin/bash
#
# inotify-hook.sh - AList 索引刷新钩子脚本
# 用于在文件系统变更后触发多个 AList 实例的索引重建
#

set -euo pipefail

export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# ========== 配置 ==========
readonly FIRMWARE_DIR="/mnt/data/firmware_os"
readonly FIRMWARE_OWNER="zun.yang:alist"
readonly INDEX_PATHS="%2FPublic"
readonly MAX_DEPTH="-1"
readonly CURL_TIMEOUT=30

# API 端点配置
readonly APIHUB_DEV_URL="https://apihub-dev.cpinnov.run/api/ops/alist-index-reflush"
readonly APIHUB_DEV_TOKEN="sqrmlnkkdmwhlkvnvjhuqmbcnu"

readonly MDM_ALIST_URL="http://10.16.18.31:32544"
readonly MDM_ALIST_USER="admin"
readonly MDM_ALIST_PASS="NUli127589%40."

readonly APIHUB_PROD_URL="https://apihub-prod.cpinnov.run/api/ops/alist-index-reflush"
readonly APIHUB_PROD_TOKEN="sqrmlnkkdmwhlkvnvjhuqmbcnu"

# ========== 函数 ==========

log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [INFO] $*"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $*" >&2
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SUCCESS] $*"
}

# 刷新通过 API Hub 代理的 AList 索引
refresh_apihub_index() {
    local name="$1"
    local url="$2"
    local token="$3"

    log_info "刷新 ${name} 索引..."

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time "${CURL_TIMEOUT}" \
        -X 'GET' \
        "${url}?paths=${INDEX_PATHS}&max_depth=${MAX_DEPTH}" \
        -H 'accept: application/json' \
        -H "Authorization: ${token}")

    if [[ "${http_code}" == "200" ]]; then
        log_success "${name} 索引刷新成功"
        return 0
    else
        log_error "${name} 索引刷新失败 (HTTP ${http_code})"
        return 1
    fi
}

# 刷新 MDM AList 索引 (需要登录获取 token)
refresh_mdm_alist_index() {
    log_info "刷新 MDM AList 索引..."

    # 获取认证 token
    local token
    token=$(curl -s --max-time "${CURL_TIMEOUT}" \
        --location --request POST \
        "${MDM_ALIST_URL}/api/auth/login?Password=${MDM_ALIST_PASS}&Username=${MDM_ALIST_USER}" \
        2>/dev/null | jq -r '.data.token // empty')

    if [[ -z "${token}" ]]; then
        log_error "MDM AList 登录失败，无法获取 token"
        return 1
    fi

    # 触发索引重建
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time "${CURL_TIMEOUT}" \
        --location "${MDM_ALIST_URL}/api/admin/index/build" \
        --header "Authorization: ${token}" \
        --form 'paths="/Public"' \
        --form 'max_depth="-1"')

    if [[ "${http_code}" == "200" ]]; then
        log_success "MDM AList 索引刷新成功"
        return 0
    else
        log_error "MDM AList 索引刷新失败 (HTTP ${http_code})"
        return 1
    fi
}

# 修复目录权限
fix_permissions() {
    log_info "修复 ${FIRMWARE_DIR} 目录权限..."

    if [[ -d "${FIRMWARE_DIR}" ]]; then
        chown "${FIRMWARE_OWNER}" -R "${FIRMWARE_DIR}"
        log_success "目录权限修复完成"
    else
        log_error "目录不存在: ${FIRMWARE_DIR}"
        return 1
    fi
}

# ========== 主流程 ==========

main() {
    local exit_code=0

    log_info "开始执行 inotify 钩子脚本"

    # 修复权限
    fix_permissions || exit_code=1

    echo

    # 刷新各 AList 实例索引
    refresh_apihub_index "alist.cpinnov.run (DEV)" "${APIHUB_DEV_URL}" "${APIHUB_DEV_TOKEN}" || exit_code=1

    echo

    refresh_mdm_alist_index || exit_code=1

    echo

    refresh_apihub_index "next.cpinnov.run (PROD)" "${APIHUB_PROD_URL}" "${APIHUB_PROD_TOKEN}" || exit_code=1

    echo

    if [[ ${exit_code} -eq 0 ]]; then
        log_success "所有任务执行完成"
    else
        log_error "部分任务执行失败"
    fi

    return ${exit_code}
}

main "$@"