//go:build linux

// Package watcher provides inotify-based file system monitoring using golang.org/x/sys/unix.
// It supports recursive directory watching and custom event handlers.
package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// EventType represents the type of file system event.
type EventType uint32

const (
	// EventCreate indicates a file or directory was created.
	EventCreate EventType = unix.IN_CREATE
	// EventModify indicates a file was modified.
	EventModify EventType = unix.IN_MODIFY
	// EventDelete indicates a file or directory was deleted.
	EventDelete EventType = unix.IN_DELETE
	// EventDeleteSelf indicates the watched directory itself was deleted.
	EventDeleteSelf EventType = unix.IN_DELETE_SELF
	// EventMovedFrom indicates a file was moved out of the watched directory.
	EventMovedFrom EventType = unix.IN_MOVED_FROM
	// EventMovedTo indicates a file was moved into the watched directory.
	EventMovedTo EventType = unix.IN_MOVED_TO
	// EventCloseWrite indicates a file opened for writing was closed.
	EventCloseWrite EventType = unix.IN_CLOSE_WRITE
	// EventAttrib indicates file attributes were changed.
	EventAttrib EventType = unix.IN_ATTRIB
	// EventIgnored indicates the watch was removed (e.g., watched dir deleted).
	EventIgnored EventType = unix.IN_IGNORED
	// EventQueueOverflow indicates the event queue overflowed.
	EventQueueOverflow EventType = unix.IN_Q_OVERFLOW
	// EventIsDir is set when the event target is a directory.
	EventIsDir EventType = unix.IN_ISDIR
)

// DefaultWatchMask monitors common file operations.
const DefaultWatchMask = unix.IN_CREATE | unix.IN_MODIFY | unix.IN_DELETE |
	unix.IN_MOVED_FROM | unix.IN_MOVED_TO | unix.IN_CLOSE_WRITE |
	unix.IN_DELETE_SELF | unix.IN_ATTRIB

// WriteCompleteWatchMask monitors only write completion events.
// Ideal for detecting when cp/rsync/scp operations finish.
// IN_CREATE is included to enable recursive watching of newly created directories.
const WriteCompleteWatchMask = unix.IN_CLOSE_WRITE | unix.IN_MOVED_TO | unix.IN_CREATE

// Event represents a file system event.
type Event struct {
	// Path is the full path of the file or directory that triggered the event.
	Path string
	// Name is the base name of the file or directory.
	Name string
	// Mask contains the raw inotify event mask.
	Mask uint32
	// Cookie is used to correlate move events.
	Cookie uint32
	// IsDir indicates whether the event target is a directory.
	IsDir bool
}

// HasType checks if the event has the specified event type.
func (e *Event) HasType(t EventType) bool {
	return e.Mask&uint32(t) != 0
}

// String returns a human-readable description of the event.
func (e *Event) String() string {
	var types []string
	if e.HasType(EventCreate) {
		types = append(types, "CREATE")
	}
	if e.HasType(EventModify) {
		types = append(types, "MODIFY")
	}
	if e.HasType(EventDelete) {
		types = append(types, "DELETE")
	}
	if e.HasType(EventDeleteSelf) {
		types = append(types, "DELETE_SELF")
	}
	if e.HasType(EventMovedFrom) {
		types = append(types, "MOVED_FROM")
	}
	if e.HasType(EventMovedTo) {
		types = append(types, "MOVED_TO")
	}
	if e.HasType(EventCloseWrite) {
		types = append(types, "CLOSE_WRITE")
	}
	if e.HasType(EventAttrib) {
		types = append(types, "ATTRIB")
	}
	if e.HasType(EventIgnored) {
		types = append(types, "IGNORED")
	}
	if e.HasType(EventQueueOverflow) {
		types = append(types, "Q_OVERFLOW")
	}
	dirStr := ""
	if e.IsDir {
		dirStr = " [DIR]"
	}
	return fmt.Sprintf("[%s]%s %s", strings.Join(types, "|"), dirStr, e.Path)
}

// EventHandler is a callback function that handles file system events.
type EventHandler func(event *Event)

// ErrorHandler is a callback function that handles errors during watching.
type ErrorHandler func(err error)

// Watcher monitors file system events using Linux inotify.
type Watcher struct {
	fd             int            // inotify file descriptor
	watches        map[int]string // watch descriptor -> path mapping
	paths          map[string]int // path -> watch descriptor mapping
	mu             sync.RWMutex   // protects watches and paths maps
	eventHandler   EventHandler   // callback for events
	errorHandler   ErrorHandler   // callback for errors
	recursive      bool           // whether to watch subdirectories
	watchMask      uint32         // inotify event mask
	done           chan struct{}  // signals watcher shutdown
	wg             sync.WaitGroup // tracks goroutines
	ignorePatterns []string       // patterns to ignore
}

// Option configures a Watcher.
type Option func(*Watcher)

// WithRecursive enables recursive watching of subdirectories.
func WithRecursive(recursive bool) Option {
	return func(w *Watcher) {
		w.recursive = recursive
	}
}

// WithWatchMask sets the inotify event mask.
func WithWatchMask(mask uint32) Option {
	return func(w *Watcher) {
		w.watchMask = mask
	}
}

// WithEventHandler sets the event handler callback.
func WithEventHandler(handler EventHandler) Option {
	return func(w *Watcher) {
		w.eventHandler = handler
	}
}

// WithErrorHandler sets the error handler callback.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(w *Watcher) {
		w.errorHandler = handler
	}
}

// WithIgnorePatterns sets patterns to ignore (e.g., "*.tmp", ".git").
func WithIgnorePatterns(patterns ...string) Option {
	return func(w *Watcher) {
		w.ignorePatterns = patterns
	}
}

// New creates a new Watcher with the given options.
func New(opts ...Option) (*Watcher, error) {
	// Initialize inotify with non-blocking flag
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, fmt.Errorf("inotify_init1 failed: %w", err)
	}

	w := &Watcher{
		fd:        fd,
		watches:   make(map[int]string),
		paths:     make(map[string]int),
		watchMask: DefaultWatchMask,
		done:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w, nil
}

// Add adds a path to the watch list.
// If recursive is enabled, all subdirectories are also watched.
func (w *Watcher) Add(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return w.addWatch(absPath)
	}

	// Add the root directory
	if err := w.addWatch(absPath); err != nil {
		return err
	}

	// If recursive, walk and add all subdirectories
	if w.recursive {
		err = filepath.Walk(absPath, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && walkPath != absPath {
				if w.shouldIgnore(walkPath) {
					return filepath.SkipDir
				}
				return w.addWatch(walkPath)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	return nil
}

// addWatch adds a single path to inotify.
func (w *Watcher) addWatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if already watching
	if _, exists := w.paths[path]; exists {
		return nil
	}

	wd, err := unix.InotifyAddWatch(w.fd, path, w.watchMask)
	if err != nil {
		return fmt.Errorf("inotify_add_watch failed for %s: %w", path, err)
	}

	w.watches[wd] = path
	w.paths[path] = wd

	return nil
}

// Remove removes a path from the watch list.
func (w *Watcher) Remove(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	wd, exists := w.paths[absPath]
	if !exists {
		return nil
	}

	_, err = unix.InotifyRmWatch(w.fd, uint32(wd))
	if err != nil && !errors.Is(err, unix.EINVAL) {
		return fmt.Errorf("inotify_rm_watch failed: %w", err)
	}

	delete(w.watches, wd)
	delete(w.paths, absPath)

	return nil
}

// Start begins watching for events. This is a blocking call.
// Call Stop() to terminate watching.
func (w *Watcher) Start() error {
	buf := make([]byte, 4096*(unix.SizeofInotifyEvent+unix.NAME_MAX+1))

	// Create epoll instance for efficient waiting
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return fmt.Errorf("epoll_create1 failed: %w", err)
	}
	defer unix.Close(epfd)

	// Add inotify fd to epoll
	event := unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(w.fd),
	}
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, w.fd, &event); err != nil {
		return fmt.Errorf("epoll_ctl failed: %w", err)
	}

	events := make([]unix.EpollEvent, 1)

	for {
		select {
		case <-w.done:
			return nil
		default:
		}

		// Wait for events with timeout for checking done channel
		n, err := unix.EpollWait(epfd, events, 100)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return fmt.Errorf("epoll_wait failed: %w", err)
		}

		if n == 0 {
			continue
		}

		// Read events from inotify
		bytesRead, err := unix.Read(w.fd, buf)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			return fmt.Errorf("read failed: %w", err)
		}

		if bytesRead < unix.SizeofInotifyEvent {
			continue
		}

		// Parse events
		w.parseEvents(buf[:bytesRead])
	}
}

// parseEvents parses raw inotify events from the buffer.
func (w *Watcher) parseEvents(buf []byte) {
	offset := 0
	for offset < len(buf) {
		if offset+unix.SizeofInotifyEvent > len(buf) {
			break
		}

		raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
		offset += unix.SizeofInotifyEvent

		// Get the name if present
		var name string
		if raw.Len > 0 {
			nameBytes := buf[offset : offset+int(raw.Len)]
			// Find null terminator
			for i, b := range nameBytes {
				if b == 0 {
					name = string(nameBytes[:i])
					break
				}
			}
			offset += int(raw.Len)
		}

		// Get the watched path
		w.mu.RLock()
		watchPath, ok := w.watches[int(raw.Wd)]
		w.mu.RUnlock()
		if !ok {
			continue
		}

		fullPath := watchPath
		if name != "" {
			fullPath = filepath.Join(watchPath, name)
		}

		// Check if should ignore
		if w.shouldIgnore(fullPath) {
			continue
		}

		isDir := raw.Mask&unix.IN_ISDIR != 0

		event := &Event{
			Path:   fullPath,
			Name:   name,
			Mask:   raw.Mask,
			Cookie: raw.Cookie,
			IsDir:  isDir,
		}

		// Handle recursive watching for new directories
		if w.recursive && isDir && raw.Mask&unix.IN_CREATE != 0 {
			if err := w.Add(fullPath); err != nil {
				if w.errorHandler != nil {
					w.errorHandler(fmt.Errorf("failed to watch new directory %s: %w", fullPath, err))
				}
			}
		}

		// Handle directory deletion
		if raw.Mask&(unix.IN_DELETE_SELF|unix.IN_IGNORED) != 0 {
			w.mu.Lock()
			if wd, exists := w.paths[fullPath]; exists {
				delete(w.watches, wd)
				delete(w.paths, fullPath)
			}
			w.mu.Unlock()
		}

		// Call event handler
		if w.eventHandler != nil {
			w.eventHandler(event)
		}
	}
}

// shouldIgnore checks if a path should be ignored based on patterns.
func (w *Watcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range w.ignorePatterns {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	close(w.done)
	w.wg.Wait()

	// Remove all watches
	w.mu.Lock()
	for wd := range w.watches {
		unix.InotifyRmWatch(w.fd, uint32(wd))
	}
	w.watches = make(map[int]string)
	w.paths = make(map[string]int)
	w.mu.Unlock()

	return unix.Close(w.fd)
}

// WatchedPaths returns a list of currently watched paths.
func (w *Watcher) WatchedPaths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.paths))
	for path := range w.paths {
		paths = append(paths, path)
	}
	return paths
}
