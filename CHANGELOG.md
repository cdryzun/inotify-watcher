# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive test suite with 80%+ coverage
- MIT License for open source compliance
- Contributing guidelines (CONTRIBUTING.md)
- Code of Conduct (CODE_OF_CONDUCT.md)
- GitHub issue templates
- Pull request template

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
