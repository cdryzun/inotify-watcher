package cmd

import (
	"testing"

	"github.com/cdryzun/inotify-watcher/watcher"
)

func TestBuildEventMask(t *testing.T) {
	tests := []struct {
		name     string
		events   []string
		expected uint32
	}{
		{
			name:     "empty",
			events:   []string{},
			expected: watcher.DefaultWatchMask,
		},
		{
			name:     "create",
			events:   []string{"create"},
			expected: uint32(watcher.EventCreate),
		},
		{
			name:     "close_write",
			events:   []string{"close_write"},
			expected: uint32(watcher.EventCloseWrite),
		},
		{
			name:     "move",
			events:   []string{"move"},
			expected: uint32(watcher.EventMovedFrom | watcher.EventMovedTo),
		},
		{
			name:     "multiple",
			events:   []string{"create", "modify", "delete"},
			expected: uint32(watcher.EventCreate | watcher.EventModify | watcher.EventDelete),
		},
		{
			name:     "with_spaces",
			events:   []string{" create ", " modify "},
			expected: uint32(watcher.EventCreate | watcher.EventModify),
		},
		{
			name:     "case_insensitive",
			events:   []string{"CREATE", "Modify", "DELETE"},
			expected: uint32(watcher.EventCreate | watcher.EventModify | watcher.EventDelete),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEventMask(tt.events)
			if result != tt.expected {
				t.Errorf("Expected mask %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetEventTypeString(t *testing.T) {
	tests := []struct {
		name     string
		event    *watcher.Event
		expected string
	}{
		{
			name: "create",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCreate),
			},
			expected: "CREATE",
		},
		{
			name: "close_write",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCloseWrite),
			},
			expected: "CLOSE_WRITE",
		},
		{
			name: "multiple",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCreate | watcher.EventModify),
			},
			expected: "CREATE|MODIFY",
		},
		{
			name: "moved_to",
			event: &watcher.Event{
				Mask: uint32(watcher.EventMovedTo),
			},
			expected: "MOVED_TO",
		},
		{
			name: "unknown",
			event: &watcher.Event{
				Mask: 0x00010000, // Unused bit
			},
			expected: "UNKNOWN(0x10000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEventTypeString(tt.event)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMatchesEventFilter(t *testing.T) {
	tests := []struct {
		name     string
		event    *watcher.Event
		filters  []string
		expected bool
	}{
		{
			name: "match_create",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCreate),
			},
			filters:  []string{"create"},
			expected: true,
		},
		{
			name: "no_match",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCreate),
			},
			filters:  []string{"delete", "modify"},
			expected: false,
		},
		{
			name: "match_one_of_multiple",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCloseWrite),
			},
			filters:  []string{"create", "close_write", "delete"},
			expected: true,
		},
		{
			name: "match_move",
			event: &watcher.Event{
				Mask: uint32(watcher.EventMovedTo),
			},
			filters:  []string{"move"},
			expected: true,
		},
		{
			name: "empty_filters",
			event: &watcher.Event{
				Mask: uint32(watcher.EventCreate),
			},
			filters:  []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesEventFilter(tt.event, tt.filters)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
