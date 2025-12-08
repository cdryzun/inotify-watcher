//go:build !linux

// Package watcher provides inotify-based file system monitoring.
// This stub file is used on non-Linux platforms where inotify is not available.
package watcher

import (
	"errors"
)

// ErrNotSupported is returned when inotify is not supported on the current platform.
var ErrNotSupported = errors.New("inotify is only supported on Linux")

// EventType represents the type of file system event.
type EventType uint32

const (
	EventCreate     EventType = 0x00000100
	EventModify     EventType = 0x00000002
	EventDelete     EventType = 0x00000200
	EventDeleteSelf EventType = 0x00000400
	EventMovedFrom  EventType = 0x00000040
	EventMovedTo    EventType = 0x00000080
	EventCloseWrite EventType = 0x00000008
	EventAttrib     EventType = 0x00000004
	EventIsDir      EventType = 0x40000000
)

// DefaultWatchMask is the default watch mask for common file operations.
const DefaultWatchMask uint32 = 0x000003CE

// WriteCompleteWatchMask monitors only write completion events.
const WriteCompleteWatchMask uint32 = 0x00000088

// Event represents a file system event.
type Event struct {
	Path   string
	Name   string
	Mask   uint32
	Cookie uint32
	IsDir  bool
}

// HasType checks if the event has the specified event type.
func (e *Event) HasType(t EventType) bool {
	return e.Mask&uint32(t) != 0
}

// String returns a human-readable description of the event.
func (e *Event) String() string {
	return e.Path
}

// EventHandler is a callback function that handles file system events.
type EventHandler func(event *Event)

// ErrorHandler is a callback function that handles errors during watching.
type ErrorHandler func(err error)

// Watcher monitors file system events using Linux inotify.
// On non-Linux platforms, this is a stub that returns ErrNotSupported.
type Watcher struct{}

// Option configures a Watcher.
type Option func(*Watcher)

// WithRecursive enables recursive watching of subdirectories.
func WithRecursive(recursive bool) Option {
	return func(w *Watcher) {}
}

// WithWatchMask sets the inotify event mask.
func WithWatchMask(mask uint32) Option {
	return func(w *Watcher) {}
}

// WithEventHandler sets the event handler callback.
func WithEventHandler(handler EventHandler) Option {
	return func(w *Watcher) {}
}

// WithErrorHandler sets the error handler callback.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(w *Watcher) {}
}

// WithIgnorePatterns sets patterns to ignore.
func WithIgnorePatterns(patterns ...string) Option {
	return func(w *Watcher) {}
}

// New creates a new Watcher. On non-Linux platforms, it returns ErrNotSupported.
func New(opts ...Option) (*Watcher, error) {
	return nil, ErrNotSupported
}

// Add adds a path to the watch list.
func (w *Watcher) Add(path string) error {
	return ErrNotSupported
}

// Remove removes a path from the watch list.
func (w *Watcher) Remove(path string) error {
	return ErrNotSupported
}

// Start begins watching for events.
func (w *Watcher) Start() error {
	return ErrNotSupported
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	return ErrNotSupported
}

// WatchedPaths returns a list of currently watched paths.
func (w *Watcher) WatchedPaths() []string {
	return nil
}
