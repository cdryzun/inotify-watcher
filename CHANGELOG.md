# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2026-03-01

### Security
- Remove `hooks/inotify-hook.sh` from entire git history; the file contained
  hardcoded production API tokens and plaintext passwords that were accidentally
  committed. All credentials have been rotated.
- Add `hooks/` to `.gitignore` to prevent future accidental commits of
  environment-specific production scripts

### Fixed
- Fix CLI binary name still showing `truenas-artifact-inotify-hook` after
  rebranding; update `Use`, help text, and `--config` flag description to
  `inotify-watcher`
- Fix config file lookup using old name `.truenas-artifact-inotify-hook.yaml`;
  now correctly loads `~/.inotify-watcher.yaml` as documented
- Fix `--events` flag not disabling write-complete mode's CREATE suppression;
  combining `--mode=write-complete --events=create` previously emitted zero events
- Remove redundant 1ms sleep in `hookDebouncer.trigger()` held while owning
  the mutex; concurrent execution is correctly guarded by the `running` flag
- Fix command injection risk in `executeHook`: validate hook path, check file
  permissions, and sanitize event parameters before execution
- Fix `Watcher.Stop()` to be idempotent; calling it multiple times no longer
  panics on double channel close
- Improve `shouldIgnore()` to report invalid glob patterns via errorHandler
  instead of silently swallowing errors
- Use `time.Local` instead of hardcoded `Asia/Shanghai` timezone in build
  time display

### Added
- Initial release
- Linux inotify-based file system monitoring using `golang.org/x/sys/unix`
- Recursive directory watching with automatic subdirectory monitoring
- Multiple directory concurrent watching support
- Two watch modes: `write-complete` (CLOSE_WRITE, MOVED_TO) and `default`
- Hook command execution on file events with debouncing mechanism
- Flexible filtering: ignore patterns, event type, file-only/directory-only
- Configuration file support (YAML) and environment variables (`INOTIFY_HOOK_`)
- systemd service integration for TrueNAS SCALE
- Cross-platform build support (amd64, arm64)

[Unreleased]: https://github.com/cdryzun/inotify-watcher/compare/v1.0.3...HEAD
[1.0.3]: https://github.com/cdryzun/inotify-watcher/releases/tag/v1.0.3
