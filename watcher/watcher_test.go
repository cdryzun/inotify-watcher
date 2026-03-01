package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	if w.fd == 0 {
		t.Error("Expected valid file descriptor")
	}
	if w.watches == nil {
		t.Error("Expected watches map to be initialized")
	}
	if w.paths == nil {
		t.Error("Expected paths map to be initialized")
	}
}

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected *Watcher
	}{
		{
			name: "recursive",
			opts: []Option{WithRecursive(true)},
			expected: &Watcher{
				recursive: true,
			},
		},
		{
			name: "watch_mask",
			opts: []Option{WithWatchMask(WriteCompleteWatchMask)},
			expected: &Watcher{
				watchMask: WriteCompleteWatchMask,
			},
		},
		{
			name: "ignore_patterns",
			opts: []Option{WithIgnorePatterns(".git", "*.tmp")},
			expected: &Watcher{
				ignorePatterns: []string{".git", "*.tmp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(tt.opts...)
			if err != nil {
				t.Fatalf("Failed to create watcher: %v", err)
			}
			defer w.Stop()

			switch tt.name {
			case "recursive":
				if w.recursive != tt.expected.recursive {
					t.Errorf("Expected recursive=%v, got %v", tt.expected.recursive, w.recursive)
				}
			case "watch_mask":
				if w.watchMask != tt.expected.watchMask {
					t.Errorf("Expected watchMask=%v, got %v", tt.expected.watchMask, w.watchMask)
				}
			case "ignore_patterns":
				if len(w.ignorePatterns) != len(tt.expected.ignorePatterns) {
					t.Errorf("Expected %d patterns, got %d", len(tt.expected.ignorePatterns), len(w.ignorePatterns))
				}
			}
		})
	}
}

func TestAdd(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	w, err := New(WithRecursive(true))
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Test adding directory
	if err := w.Add(tmpDir); err != nil {
		t.Errorf("Failed to add directory: %v", err)
	}

	paths := w.WatchedPaths()
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 watched paths (root + subdir), got %d", len(paths))
	}
}

func TestAddFile(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "watcher-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Test adding file
	if err := w.Add(tmpFile.Name()); err != nil {
		t.Errorf("Failed to add file: %v", err)
	}

	paths := w.WatchedPaths()
	if len(paths) != 1 {
		t.Errorf("Expected 1 watched path, got %d", len(paths))
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		expected bool
	}{
		{
			name:     "ignore_git_dir",
			patterns: []string{".git"},
			path:     "/tmp/test/.git",
			expected: true,
		},
		{
			name:     "ignore_tmp_files",
			patterns: []string{"*.tmp"},
			path:     "/tmp/test/file.tmp",
			expected: true,
		},
		{
			name:     "allow_normal_files",
			patterns: []string{".git", "*.tmp"},
			path:     "/tmp/test/file.txt",
			expected: false,
		},
		{
			name:     "ignore_swp_files",
			patterns: []string{"*.swp"},
			path:     "/tmp/test/.file.swp",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(WithIgnorePatterns(tt.patterns...))
			if err != nil {
				t.Fatalf("Failed to create watcher: %v", err)
			}
			defer w.Stop()

			result := w.shouldIgnore(tt.path)
			if result != tt.expected {
				t.Errorf("Expected shouldIgnore=%v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEventHasType(t *testing.T) {
	event := &Event{
		Mask: uint32(EventCreate | EventCloseWrite),
	}

	if !event.HasType(EventCreate) {
		t.Error("Expected event to have CREATE type")
	}
	if !event.HasType(EventCloseWrite) {
		t.Error("Expected event to have CLOSE_WRITE type")
	}
	if event.HasType(EventDelete) {
		t.Error("Expected event not to have DELETE type")
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		expected string
	}{
		{
			name: "create_dir",
			event: &Event{
				Mask:  uint32(EventCreate | EventIsDir),
				Path:  "/tmp/test",
				IsDir: true,
			},
			expected: "[CREATE] [DIR] /tmp/test",
		},
		{
			name: "close_write_file",
			event: &Event{
				Mask:  uint32(EventCloseWrite),
				Path:  "/tmp/test.txt",
				IsDir: false,
			},
			expected: "[CLOSE_WRITE] /tmp/test.txt",
		},
		{
			name: "multiple_events",
			event: &Event{
				Mask:  uint32(EventCreate | EventModify),
				Path:  "/tmp/test.txt",
				IsDir: false,
			},
			expected: "[CREATE|MODIFY] /tmp/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Add and then remove
	if err := w.Add(tmpDir); err != nil {
		t.Fatalf("Failed to add directory: %v", err)
	}

	if err := w.Remove(tmpDir); err != nil {
		t.Errorf("Failed to remove directory: %v", err)
	}

	paths := w.WatchedPaths()
	if len(paths) != 0 {
		t.Errorf("Expected 0 watched paths after remove, got %d", len(paths))
	}
}

func TestWatchedPaths(t *testing.T) {
	// Create temp directories
	tmpDir1, err := os.MkdirTemp("", "watcher-test-1-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "watcher-test-2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	// Add multiple paths
	if err := w.Add(tmpDir1); err != nil {
		t.Fatalf("Failed to add dir1: %v", err)
	}
	if err := w.Add(tmpDir2); err != nil {
		t.Fatalf("Failed to add dir2: %v", err)
	}

	paths := w.WatchedPaths()
	if len(paths) != 2 {
		t.Errorf("Expected 2 watched paths, got %d", len(paths))
	}
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "watcher-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	events := make(chan *Event, 10)

	w, err := New(
		WithRecursive(true),
		WithWatchMask(WriteCompleteWatchMask),
		WithEventHandler(func(event *Event) {
			events <- event
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := w.Add(tmpDir); err != nil {
		t.Fatalf("Failed to add directory: %v", err)
	}

	// Start watcher in goroutine
	go w.Start()
	defer w.Stop()

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Wait for event
	select {
	case event := <-events:
		if event.Path != testFile {
			t.Errorf("Expected event path %s, got %s", testFile, event.Path)
		}
		// Accept either CLOSE_WRITE or CREATE (WriteCompleteWatchMask includes CREATE for recursive watching)
		if !event.HasType(EventCloseWrite) && !event.HasType(EventCreate) {
			t.Errorf("Expected CLOSE_WRITE or CREATE event, got %v", event)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for event")
	}
}
