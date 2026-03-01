# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.2] - 2026-03-01

### Fixed
- Fix CLI binary name still showing `truenas-artifact-inotify-hook` after
  rebranding; update `Use`, help text, and `--config` flag description to
  `inotify-watcher` (#2)
- Fix config file lookup using old name `.truenas-artifact-inotify-hook.yaml`;
  now correctly loads `~/.inotify-watcher.yaml` as documented (#2)
- Fix `--events` flag not disabling write-complete mode's CREATE suppression;
  combining `--mode=write-complete --events=create` previously emitted zero
  events — `isWriteCompleteMode` is now reset when `--events` is provided (#2)

## [1.0.1] - 2026-03-01

### Fixed
- Remove redundant 1ms sleep in `hookDebouncer.trigger()` that was held
  while owning the mutex, making it impossible for `execute()` to advance.
  Concurrent execution is correctly guarded by the `running` flag (#1)
- Fix command injection risk in `executeHook`: validate hook path, check
  file permissions, and sanitize event parameters before execution
- Fix `Watcher.Stop()` to be idempotent; calling it multiple times no
  longer panics on double channel close
- Fix timer race in `hookDebouncer` by relying on `running` flag as the
  sole concurrency guard
- Improve `shouldIgnore()` to report invalid glob patterns via errorHandler
  instead of silently swallowing errors
- Use `time.Local` instead of hardcoded `Asia/Shanghai` timezone in
  build time display

## [1.0.0] - 2025-12-08

### Added
- Initial release
- Linux inotify-based file system monitoring using `golang.org/x/sys/unix`
- Recursive directory watching with automatic subdirectory monitoring
- Multiple directory concurrent watching support
- Two watch modes:
  - `write-complete`: Monitor only CLOSE_WRITE and MOVED_TO events (default)
  - `default`: Monitor all common file system events
- Hook command execution on file events
- Hook debouncing mechanism to prevent excessive executions
- Flexible filtering options:
  - Ignore patterns (glob patterns like `.git`, `*.tmp`)
  - Event type filtering
  - File-only or directory-only filtering
- Configuration file support (YAML)
- Environment variable support (prefix `INOTIFY_HOOK_`)
- systemd service integration for TrueNAS SCALE
- Deployment automation script (`deploy/install.sh`)
- Cross-platform build support (amd64, arm64)
- Taskfile-based build automation

### Features
- **Direct inotify API**: Uses `golang.org/x/sys/unix` system calls without third-party wrappers
- **Write completion detection**: Default mode captures `cp`/`rsync`/`scp` completion timing
- **Concurrent path addition**: Parallel directory watching for faster initialization
- **Automatic directory watching**: New directories are automatically watched in recursive mode
- **Graceful shutdown**: Clean resource cleanup on SIGINT/SIGTERM
- **Memory efficient**: Uses epoll for event waiting with minimal overhead

### Documentation
- Comprehensive README with usage examples
- Systemd service deployment guide
- Hook script examples
- Troubleshooting guide

### Supported Platforms
- Linux (amd64, arm64)
- Tested on TrueNAS SCALE (Debian-based)

[Unreleased]: https://github.com/your-org/truenas-artifact-inotify-hook/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/your-org/truenas-artifact-inotify-hook/releases/tag/v1.0.0
