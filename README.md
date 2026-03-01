# inotify-watcher

[![GitHub release](https://img.shields.io/github/release/cdryzun/inotify-watcher.svg?style=flat-square)](https://github.com/cdryzun/inotify-watcher/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/cdryzun/inotify-watcher?style=flat-square)](https://goreportcard.com/report/github.com/cdryzun/inotify-watcher)
[![CI](https://github.com/cdryzun/inotify-watcher/workflows/CI/badge.svg?style=flat-square)](https://github.com/cdryzun/inotify-watcher/actions)
[![codecov](https://codecov.io/gh/cdryzun/inotify-watcher/branch/main/graph/badge.svg?style=flat-square)](https://codecov.io/gh/cdryzun/inotify-watcher)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/dl/)
[![Platform](https://img.shields.io/badge/Platform-Linux%20amd64%20%7C%20arm64-lightgrey?style=flat-square)](https://github.com/cdryzun/inotify-watcher/releases)

🚀 **High-performance Linux inotify file system watcher** - Designed for TrueNAS SCALE with write-completion detection, automatic hook triggers, and intelligent debouncing

**[中文文档](README_CN.md)** | **English**

A high-performance file system monitoring tool built on Linux inotify using `golang.org/x/sys/unix` for low-level system calls. Specifically designed for artifact directory monitoring on TrueNAS SCALE, providing precise file operation capture capabilities.

## ✨ Key Features

- 🎯 **Precise Write Detection** - Monitors `CLOSE_WRITE` and `MOVED_TO` events by default, accurately capturing `cp`/`rsync`/`scp` completion timing
- ⚡ **High-Performance Concurrency** - Concurrent multi-directory watching with 3-5x faster initialization
- 🔄 **Automatic Recursive Watching** - Automatically watches all subdirectories; newly created directories are auto-added
- 🛡️ **Intelligent Debouncing** - Aggregates rapid events to prevent excessive hook triggers
- 🎨 **Flexible Configuration** - Supports ignore patterns, event type filtering, file/directory filtering, YAML config, and environment variables
- 📦 **Zero Dependencies** - Direct use of `golang.org/x/sys/unix` system calls, no third-party wrappers
- 🚀 **Production Ready** - 77%+ test coverage, complete CI/CD, running in TrueNAS production environments

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Examples](#examples)
- [Performance](#performance)
- [Comparison](#comparison)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## 🚀 Quick Start

### One-Line Installation

```bash
# Download latest release (replace X.Y.Z with the latest version)
wget https://github.com/cdryzun/inotify-watcher/releases/latest/download/inotify-watcher-linux-amd64.tar.gz
tar -xzf inotify-watcher-linux-amd64.tar.gz
sudo mv inotify-watcher /usr/local/bin/inotify-hook
sudo chmod +x /usr/local/bin/inotify-hook

# Verify installation
inotify-hook version
```

### 5-Second Setup

```bash
# Watch a directory and execute script on file upload completion
inotify-hook watch /mnt/data/artifacts \
  --mode=write-complete \
  --hook=/usr/local/bin/on-file-ready.sh

# View real-time events (debug mode)
inotify-hook watch /mnt/data/artifacts --verbose
```

## 📦 Installation

### Build from Source

```bash
# Clone repository
git clone https://github.com/cdryzun/inotify-watcher.git
cd inotify-watcher

# Build using Task (recommended)
task build:truenas

# Or build manually
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/inotify-watcher-linux-amd64 .
```

### Deploy to TrueNAS

```bash
scp dist/inotify-watcher-linux-amd64 root@truenas:/usr/local/bin/inotify-hook
ssh root@truenas chmod +x /usr/local/bin/inotify-hook
```

## Usage

### Basic Usage

```bash
# Watch a directory (default: write-complete mode)
inotify-hook watch /mnt/data/artifacts

# Watch multiple directories
inotify-hook watch /mnt/data/artifacts /mnt/data/uploads
```

### Watch Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `write-complete` | Monitors only write completion (CLOSE_WRITE, MOVED_TO) | **Default**, ideal for cp/rsync/scp completion detection |
| `default` | Monitors all common events | Requires complete file operation logging |

```bash
# Write-complete mode (default)
inotify-hook watch /mnt/data/artifacts --mode=write-complete

# Full event mode
inotify-hook watch /mnt/data/artifacts --mode=default
```

### Hook Commands

Execute custom scripts when file events occur:

```bash
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/on-artifact-ready.sh
```

Hook scripts receive these parameters:
- `$1`: Event type (CLOSE_WRITE, MOVED_TO, CREATE, DELETE, etc.)
- `$2`: Full file path
- `$3`: File name
- `$4`: Is directory (true/false)

Example hook script:

```bash
#!/bin/bash
# /usr/local/bin/on-artifact-ready.sh

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

echo "[$(date)] Event: $EVENT_TYPE, Path: $FILE_PATH, IsDir: $IS_DIR"

# Example: Trigger CI/CD pipeline
if [[ "$FILE_PATH" == *.tar.gz ]]; then
    curl -X POST "https://ci.example.com/trigger" \
        -d "artifact=$FILE_PATH"
fi
```

### Hook Debouncing

When using `rsync`, `cp -r`, or other bulk file operations, many events are triggered in a short time. The debouncing mechanism aggregates these events and executes the hook only once after operations complete.

```bash
# Default 500ms debounce window
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh

# Large file transfers, use 1 second debounce
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh --debounce=1000

# Bulk operations, use 2 second debounce
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/sync.sh --debounce=2000

# Disable debouncing, trigger hook on every event
inotify-hook watch /mnt/data/artifacts --hook=/usr/local/bin/notify.sh --debounce=0
```

**How it works:**
1. Starts a timer when an event is received (default 500ms)
2. Resets the timer if new events arrive before it expires
3. Executes the hook using the last event's information when timer expires
4. Logs show the number of aggregated events

**Recommended values:**

| Scenario | Recommended | Description |
|----------|-------------|-------------|
| General use | 500ms | Default, suitable for most scenarios |
| Large file transfers | 1000ms | rsync/scp large files |
| Bulk copying | 2000ms | cp -r many small files |
| Real-time notifications | 0 | Disable debounce, trigger on every event |

### Filtering Options

```bash
# Watch files only (ignore directory events)
inotify-hook watch /mnt/data/artifacts --files-only

# Watch directories only (ignore file events)
inotify-hook watch /mnt/data/artifacts --dirs-only

# Ignore specific patterns
inotify-hook watch /mnt/data/artifacts --ignore=".git,*.tmp,*.swp"

# Watch specific events
inotify-hook watch /mnt/data/artifacts --events=close_write,moved_to

# Non-recursive watching
inotify-hook watch /mnt/data/artifacts --recursive=false
```

### Configuration File

Create `~/.inotify-watcher.yaml`:

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
  debounce: 500          # Hook debounce time (milliseconds), 0 to disable
  files-only: false
  dirs-only: false
```

Also supports environment variables (prefix `INOTIFY_HOOK_`):

```bash
export INOTIFY_HOOK_WATCH_MODE=write-complete
export INOTIFY_HOOK_WATCH_HOOK=/usr/local/bin/on-artifact-ready.sh
export INOTIFY_HOOK_WATCH_DEBOUNCE=1000
```

## Examples

Check the [examples/](examples/) directory for complete hook script examples:

- [CI/CD Trigger](examples/ci-cd-trigger.sh) - Trigger Jenkins/GitLab CI pipelines
- [AList Sync](examples/alist-sync.sh) - Refresh AList indexes
- [Notifications](examples/notifications.sh) - Multi-channel notifications (Slack, Email)
- [Backup Files](examples/backup-files.sh) - Automatic file backup with versioning

## ⚡ Performance

### Performance Advantages

| Feature | Benefit | Data |
|---------|---------|------|
| **Concurrent initialization** | Parallel directory watching | 3-5x speed boost |
| **Event debouncing** | Intelligent event aggregation | 90% reduction in hook calls |
| **Memory footprint** | Efficient epoll-based waiting | < 10MB resident memory |
| **Zero-copy** | Direct system calls | No third-party library overhead |

### Resource Usage

```
Memory:    ~10 MB (watching 1000+ directories)
CPU:       < 1% (when idle)
File Descriptors: ~1000 (watching 1000 directories)
```

### Performance Tips

1. **Debounce settings**:
   - Large file transfers: `--debounce=1000` (1 second)
   - Bulk operations: `--debounce=2000` (2 seconds)
   - Real-time response: `--debounce=0` (disable)

2. **Concurrent watching**:
   ```bash
   # Automatically adds multiple directories concurrently
   inotify-hook watch /dir1 /dir2 /dir3
   ```

3. **Filter optimization**:
   ```bash
   # Only watch needed file types to reduce event processing
   inotify-hook watch /data --files-only --ignore=".git,*.tmp"
   ```

## 🔍 Comparison with Other Tools

| Feature | inotify-watcher | inotify-tools | fsnotify | entr |
|---------|----------------|---------------|----------|------|
| **Write-completion detection** | ✅ Native support | ❌ Requires script | ❌ Requires code | ⚠️ Limited |
| **Hook debouncing** | ✅ Built-in | ❌ None | ❌ None | ❌ None |
| **Multi-directory watching** | ✅ Concurrent | ⚠️ Multi-process | ✅ Supported | ❌ Single directory |
| **Recursive watching** | ✅ Automatic | ⚠️ Requires script | ✅ Supported | ❌ Not supported |
| **Configuration file** | ✅ YAML | ❌ None | ⚠️ Code config | ❌ None |
| **Systemd integration** | ✅ Complete | ⚠️ DIY required | ❌ None | ❌ None |
| **Test coverage** | ✅ 77%+ | ❓ Unknown | ✅ High | ❓ Unknown |
| **Performance** | ⚡ High | ⚡ High | ⚡ Medium | ⚡ High |
| **Ease of use** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |

**Use cases:**

- **inotify-watcher**: Production environments, NAS monitoring, CI/CD triggers
- **inotify-tools**: Simple scripts, quick prototyping
- **fsnotify**: Go application integration, library usage
- **entr**: Command-line tools, development-time watching

## ❓ FAQ

<details>
<summary><b>1. How to test hook scripts?</b></summary>

```bash
# Manually test hook script
/usr/local/bin/your-hook.sh CLOSE_WRITE /tmp/test.txt test.txt false

# Use verbose mode to view events
inotify-hook watch /data --verbose
```
</details>

<details>
<summary><b>2. Why am I not receiving events?</b></summary>

**Checklist:**
1. Confirm directory exists and has proper permissions
2. Check if ignore patterns are filtering the files
3. Use `--verbose` to view raw events
4. Confirm filesystem supports inotify (NFS may have issues)

```bash
# Debug mode
inotify-hook watch /data --verbose --events=all
```
</details>

<details>
<summary><b>3. Hook executes too frequently, what to do?</b></summary>

Use the `--debounce` parameter:

```bash
# 1 second debounce
inotify-hook watch /data --hook=/script.sh --debounce=1000

# Disable debounce (trigger on every event)
inotify-hook watch /data --hook=/script.sh --debounce=0
```
</details>

<details>
<summary><b>4. How to watch remote filesystems?</b></summary>

inotify only supports local filesystems. For NFS/SMB:
- **Option 1**: Run inotify-hook on the server side
- **Option 2**: Use polling methods (e.g., `ls` + diff)
- **Option 3**: Use filesystem-specific events (e.g., NFS callbacks)

</details>

<details>
<summary><b>5. Does it support macOS/Windows?</b></summary>

❌ No. This tool is specifically designed for Linux inotify.

**Alternatives:**
- **macOS**: Use [fswatch](https://github.com/emcrisostomo/fswatch)
- **Windows**: Use [fsnotify](https://github.com/fsnotify/fsnotify) (Go library)
- **Cross-platform**: Consider refactoring to use fsnotify library

</details>

## 🤝 Contributing

We welcome all forms of contributions!

### Quick Contribution

- 🐛 [Report Bug](https://github.com/cdryzun/inotify-watcher/issues/new?template=bug_report.md)
- 💡 [Request Feature](https://github.com/cdryzun/inotify-watcher/issues/new?template=feature_request.md)
- 📖 Improve documentation
- 🔧 Submit PR

### Contribution Process

See [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Code standards
- Commit conventions
- PR process
- Development environment setup

### Contributors

Thanks to all contributors!

<a href="https://github.com/cdryzun/inotify-watcher/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=cdryzun/inotify-watcher" />
</a>

## 📜 License

MIT License - See [LICENSE](LICENSE)

## 🌟 Star History

If this project helps you, please give it a ⭐ Star!

[![Star History Chart](https://api.star-history.com/svg?repos=cdryzun/inotify-watcher&type=Date)](https://star-history.com/#cdryzun/inotify-watcher&Date)

## 📞 Community & Support

- 💬 [GitHub Discussions](https://github.com/cdryzun/inotify-watcher/discussions) - Questions and discussions
- 🐛 [GitHub Issues](https://github.com/cdryzun/inotify-watcher/issues) - Bug reports
- 📧 Email: cdryzun@users.noreply.github.com

## 🔗 Related Projects

- [inotify-tools](https://github.com/inotify-tools/inotify-tools) - C-based inotify toolset
- [fsnotify](https://github.com/fsnotify/fsnotify) - Go cross-platform filesystem notification library
- [entr](http://eradman.com/entrproject/) - Run commands when files change

---

**Made with ❤️ by [cdryzun](https://github.com/cdryzun)**

**If this project helps you, please consider giving it a ⭐ Star to support development!**
