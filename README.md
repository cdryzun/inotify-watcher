# TrueNAS Artifact Inotify Hook

基于 Linux inotify 的文件系统监听工具，使用 `golang.org/x/sys/unix` 实现底层系统调用。专为 TrueNAS SCALE 上的 artifact 目录监控设计。

## 功能特性

- **底层 inotify API**: 直接使用 `golang.org/x/sys/unix` 系统调用，无第三方封装
- **写入完成检测**: 默认监听 `CLOSE_WRITE` 和 `MOVED_TO` 事件，精准捕获 `cp`/`rsync`/`scp` 完成时机
- **递归目录监听**: 自动监听所有子目录，新建目录自动添加监听
- **多目录并发监听**: 支持同时监听多个目录，并发添加提升初始化速度
- **Hook 命令执行**: 文件变化时触发自定义脚本
- **Hook 防抖机制**: 聚合短时间内的多个事件，避免 hook 被频繁触发
- **灵活过滤**: 支持忽略模式、事件类型过滤、文件/目录过滤
- **配置文件支持**: 支持 YAML 配置和环境变量

## 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://gitlab.cpinnov.run/devops/truenas-artifact-inotify-hook.git
cd truenas-artifact-inotify-hook

# 使用 Task 构建（推荐）
task build:truenas

# 或手动构建
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/truenas-artifact-inotify-hook-linux-amd64 .
```

### 部署到 TrueNAS

```bash
scp dist/truenas-artifact-inotify-hook-linux-amd64 root@truenas:/usr/local/bin/inotify-hook
ssh root@truenas chmod +x /usr/local/bin/inotify-hook
```

## 使用方法

### 基本用法

```bash
# 监听目录（默认 write-complete 模式）
inotify-hook watch /mnt/data/artifacts

# 监听多个目录
inotify-hook watch /mnt/data/artifacts /mnt/data/uploads
```

### 监听模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `write-complete` | 仅监听写入完成 (CLOSE_WRITE, MOVED_TO) | **默认**，适合 cp/rsync/scp 完成检测 |
| `default` | 监听所有常见事件 | 需要完整文件操作日志 |

```bash
# 写入完成模式（默认）
inotify-hook watch /mnt/data/artifacts --mode=write-complete

# 完整事件模式
inotify-hook watch /mnt/data/artifacts --mode=default
```

### Hook 命令

当文件事件发生时，执行指定的脚本：

```bash
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/on-artifact-ready.sh
```

Hook 脚本接收以下参数：
- `$1`: 事件类型 (CLOSE_WRITE, MOVED_TO, CREATE, DELETE, etc.)
- `$2`: 文件完整路径
- `$3`: 文件名
- `$4`: 是否为目录 (true/false)

示例 Hook 脚本：

```bash
#!/bin/bash
# /usr/local/bin/on-artifact-ready.sh

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

echo "[$(date)] Event: $EVENT_TYPE, Path: $FILE_PATH, IsDir: $IS_DIR"

# 示例：触发 CI/CD 流水线
if [[ "$FILE_PATH" == *.tar.gz ]]; then
    curl -X POST "https://ci.example.com/trigger" \
        -d "artifact=$FILE_PATH"
fi
```

### Hook 防抖机制

当使用 `rsync`、`cp -r` 等批量文件操作时，会在短时间内触发大量事件。防抖机制可以将这些事件聚合，只在操作完成后执行一次 hook。

```bash
# 默认 500ms 防抖窗口
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh

# 大文件传输场景，使用 1 秒防抖
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh --debounce=1000

# 批量操作场景，使用 2 秒防抖
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh --debounce=2000

# 禁用防抖，每个事件都触发 hook
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/notify.sh --debounce=0
```

**工作原理：**
1. 收到事件后启动计时器（默认 500ms）
2. 在计时器到期前如有新事件，重置计时器
3. 计时器到期后，使用最后一个事件的信息执行 hook
4. 日志会显示聚合的事件数量

**推荐值：**

| 场景 | 推荐值 | 说明 |
|------|--------|------|
| 一般使用 | 500ms | 默认值，适合大多数场景 |
| 大文件传输 | 1000ms | rsync/scp 大文件 |
| 批量复制 | 2000ms | cp -r 大量小文件 |
| 实时通知 | 0 | 禁用防抖，每个事件都触发 |

### 过滤选项

```bash
# 仅监听文件（忽略目录事件）
inotify-hook watch /mnt/data/artifacts --files-only

# 仅监听目录（忽略文件事件）
inotify-hook watch /mnt/data/artifacts --dirs-only

# 忽略特定模式
inotify-hook watch /mnt/data/artifacts --ignore=".git,*.tmp,*.swp"

# 监听特定事件
inotify-hook watch /mnt/data/artifacts --events=close_write,moved_to

# 非递归监听
inotify-hook watch /mnt/data/artifacts --recursive=false
```

### 配置文件

创建 `~/.truenas-artifact-inotify-hook.yaml`:

```yaml
verbose: false

watch:
  mode: write-complete
  recursive: true
  ignore:
    - ".git"
    - "*.tmp"
    - "*.swp"
    - "*~"
  hook: /usr/local/bin/on-artifact-ready.sh
  debounce: 500          # hook 防抖时间 (毫秒)，0 禁用
  files-only: false
  dirs-only: false
```

也可使用环境变量（前缀 `INOTIFY_HOOK_`）：

```bash
export INOTIFY_HOOK_WATCH_MODE=write-complete
export INOTIFY_HOOK_WATCH_HOOK=/usr/local/bin/on-artifact-ready.sh
export INOTIFY_HOOK_WATCH_DEBOUNCE=1000
```

## 实际使用场景

### 监听 ZFS Snapshot 复制

```bash
# 监听从 ZFS snapshot 复制固件的完成事件
inotify-hook watch /mnt/data/firmware_os/prod \
  --mode=write-complete \
  --files-only \
  --hook=/usr/local/bin/notify-firmware-ready.sh
```

### 多目录监听

同时监听多个目录，并发添加提升初始化速度：

```bash
# 监听多个环境的 artifact 目录
inotify-hook watch \
  /mnt/data/firmware_os/prod/prod-os/ \
  /mnt/data/firmware_os/pre/os-pre/ \
  /mnt/data/firmware_os/test/os-test/ \
  --mode=write-complete \
  --hook=/usr/local/bin/sync-alist-index.sh \
  --debounce=1000
```

输出示例：
```
2025/12/08 10:00:00 Mode: write-complete (CLOSE_WRITE, MOVED_TO)
2025/12/08 10:00:00 Adding 3 paths to watch...
2025/12/08 10:00:00 Hook debounce: 1s
2025/12/08 10:00:01 Watching: /mnt/data/firmware_os/prod/prod-os/
2025/12/08 10:00:01 Watching: /mnt/data/firmware_os/pre/os-pre/
2025/12/08 10:00:01 Watching: /mnt/data/firmware_os/test/os-test/
2025/12/08 10:00:01 Total directories being watched: 150
2025/12/08 10:00:01 Recursive mode: enabled
2025/12/08 10:00:01 Hook command: /usr/local/bin/sync-alist-index.sh
2025/12/08 10:00:01 File watcher started. Press Ctrl+C to stop.
```

### 作为 systemd 服务运行

创建 `/etc/systemd/system/inotify-hook.service`:

```ini
[Unit]
Description=TrueNAS Artifact Inotify Hook
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/inotify-hook watch \
    /mnt/data/firmware_os/prod \
    /mnt/data/firmware_os/pre \
    /mnt/data/firmware_os/test \
    --hook=/usr/local/bin/on-artifact-ready.sh \
    --debounce=1000
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
systemctl daemon-reload
systemctl enable --now inotify-hook
```

## 命令参考

```
Usage:
  truenas-artifact-inotify-hook watch [paths...] [flags]

Flags:
      --debounce int     Hook 防抖窗口 (毫秒)，聚合事件后执行一次 hook (默认 500，0=禁用)
  -d, --dirs-only        仅报告目录事件
  -e, --events strings   过滤特定事件 (create,modify,delete,move,close_write,attrib)
  -f, --files-only       仅报告文件事件
  -h, --help             帮助信息
  -H, --hook string      事件触发时执行的命令
  -i, --ignore strings   忽略模式 (默认 [.git,*.tmp,*.swp,*~])
  -m, --mode string      监听模式: default, write-complete (默认 "write-complete")
  -r, --recursive        递归监听子目录 (默认 true)

Global Flags:
      --config string   配置文件路径
  -v, --verbose         详细输出
```

## 事件类型

| 事件 | 说明 |
|------|------|
| `CLOSE_WRITE` | 文件写入完成并关闭 |
| `MOVED_TO` | 文件移动到监听目录 |
| `CREATE` | 文件/目录创建 |
| `MODIFY` | 文件内容修改 |
| `DELETE` | 文件/目录删除 |
| `ATTRIB` | 属性变更 |

## 构建任务

使用 [Task](https://taskfile.dev/) 进行构建：

```bash
# 查看所有任务
task --list

# 为 TrueNAS 构建
task build:truenas

# 构建所有平台
task build:all

# 运行测试
task test

# 清理构建产物
task clean
```

## 系统要求

- **运行环境**: Linux (TrueNAS SCALE 基于 Debian)
- **构建环境**: Go 1.21+
- **架构支持**: amd64, arm64

## License

MIT License
