package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.cpinnov.run/devops/truenas-artifact-inotify-hook/watcher"
)

// watchCmd represents the watch command.
var watchCmd = &cobra.Command{
	Use:   "watch [paths...]",
	Short: "Watch directories for file system events",
	Long: `Watch one or more directories for file system events using inotify.

Supports recursive watching of subdirectories and can execute hook commands
when events occur. Events include file creation, modification, deletion,
and move operations.`,
	Example: `  # Watch a single directory
  truenas-artifact-inotify-hook watch /data/artifacts

  # Watch multiple directories
  truenas-artifact-inotify-hook watch /data/artifacts /data/uploads

  # Watch with a hook command
  truenas-artifact-inotify-hook watch /data/artifacts --hook=/usr/local/bin/on-change.sh

  # Watch specific events only
  truenas-artifact-inotify-hook watch /data/artifacts --events=create,close_write

  # Non-recursive watch
  truenas-artifact-inotify-hook watch /data/artifacts --recursive=false`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Watch command flags
	watchCmd.Flags().BoolP("recursive", "r", true, "Watch directories recursively")
	watchCmd.Flags().StringSliceP("ignore", "i", []string{".git", "*.tmp", "*.swp", "*~"}, "Patterns to ignore")
	watchCmd.Flags().StringP("hook", "H", "", "Command to execute on file events")
	watchCmd.Flags().StringSliceP("events", "e", []string{}, "Filter specific events (create,modify,delete,move,close_write,attrib)")

	// Bind to viper
	viper.BindPFlag("watch.recursive", watchCmd.Flags().Lookup("recursive"))
	viper.BindPFlag("watch.ignore", watchCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("watch.hook", watchCmd.Flags().Lookup("hook"))
	viper.BindPFlag("watch.events", watchCmd.Flags().Lookup("events"))
}

func runWatch(cmd *cobra.Command, args []string) error {
	paths := args
	recursive := viper.GetBool("watch.recursive")
	ignorePatterns := viper.GetStringSlice("watch.ignore")
	hookCommand := viper.GetString("watch.hook")
	eventFilter := viper.GetStringSlice("watch.events")
	verbose := viper.GetBool("verbose")

	// Build event mask if specific events are requested
	var watchMask uint32 = watcher.DefaultWatchMask
	if len(eventFilter) > 0 {
		watchMask = buildEventMask(eventFilter)
	}

	// Create event handler
	eventHandler := createEventHandler(hookCommand, eventFilter, verbose)

	// Create error handler
	errorHandler := func(err error) {
		log.Printf("Watcher error: %v", err)
	}

	// Create watcher
	w, err := watcher.New(
		watcher.WithRecursive(recursive),
		watcher.WithWatchMask(watchMask),
		watcher.WithEventHandler(eventHandler),
		watcher.WithErrorHandler(errorHandler),
		watcher.WithIgnorePatterns(ignorePatterns...),
	)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Add paths to watch
	for _, path := range paths {
		if err := w.Add(path); err != nil {
			return fmt.Errorf("failed to watch path %s: %w", path, err)
		}
		log.Printf("Watching: %s", path)
	}

	if recursive {
		log.Printf("Recursive mode: enabled")
	}
	if len(ignorePatterns) > 0 && verbose {
		log.Printf("Ignore patterns: %v", ignorePatterns)
	}
	if hookCommand != "" {
		log.Printf("Hook command: %s", hookCommand)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watcher in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- w.Start()
	}()

	log.Println("File watcher started. Press Ctrl+C to stop.")

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down...", sig)
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("watcher error: %w", err)
		}
	}

	// Stop watcher
	if err := w.Stop(); err != nil {
		log.Printf("Error stopping watcher: %v", err)
	}

	log.Println("File watcher stopped.")
	return nil
}

func buildEventMask(events []string) uint32 {
	var mask uint32
	for _, event := range events {
		switch strings.ToLower(strings.TrimSpace(event)) {
		case "create":
			mask |= uint32(watcher.EventCreate)
		case "modify":
			mask |= uint32(watcher.EventModify)
		case "delete":
			mask |= uint32(watcher.EventDelete)
		case "delete_self":
			mask |= uint32(watcher.EventDeleteSelf)
		case "move", "moved_from":
			mask |= uint32(watcher.EventMovedFrom)
		case "moved_to":
			mask |= uint32(watcher.EventMovedTo)
		case "close_write":
			mask |= uint32(watcher.EventCloseWrite)
		case "attrib":
			mask |= uint32(watcher.EventAttrib)
		}
	}
	if mask == 0 {
		return watcher.DefaultWatchMask
	}
	return mask
}

func createEventHandler(hookCommand string, eventFilter []string, verbose bool) watcher.EventHandler {
	return func(event *watcher.Event) {
		eventType := getEventTypeString(event)

		// Filter events if specified
		if len(eventFilter) > 0 && !matchesEventFilter(event, eventFilter) {
			return
		}

		// Log the event
		if verbose {
			log.Printf("Event: %s", event.String())
		} else {
			log.Printf("[%s] %s", eventType, event.Path)
		}

		// Execute hook command if configured
		if hookCommand != "" {
			go executeHook(hookCommand, event, eventType)
		}
	}
}

func matchesEventFilter(event *watcher.Event, filters []string) bool {
	for _, filter := range filters {
		switch strings.ToLower(strings.TrimSpace(filter)) {
		case "create":
			if event.HasType(watcher.EventCreate) {
				return true
			}
		case "modify":
			if event.HasType(watcher.EventModify) {
				return true
			}
		case "delete":
			if event.HasType(watcher.EventDelete) {
				return true
			}
		case "delete_self":
			if event.HasType(watcher.EventDeleteSelf) {
				return true
			}
		case "move", "moved_from":
			if event.HasType(watcher.EventMovedFrom) {
				return true
			}
		case "moved_to":
			if event.HasType(watcher.EventMovedTo) {
				return true
			}
		case "close_write":
			if event.HasType(watcher.EventCloseWrite) {
				return true
			}
		case "attrib":
			if event.HasType(watcher.EventAttrib) {
				return true
			}
		}
	}
	return false
}

func getEventTypeString(event *watcher.Event) string {
	switch {
	case event.HasType(watcher.EventCreate):
		return "CREATE"
	case event.HasType(watcher.EventCloseWrite):
		return "CLOSE_WRITE"
	case event.HasType(watcher.EventModify):
		return "MODIFY"
	case event.HasType(watcher.EventDelete):
		return "DELETE"
	case event.HasType(watcher.EventDeleteSelf):
		return "DELETE_SELF"
	case event.HasType(watcher.EventMovedFrom):
		return "MOVED_FROM"
	case event.HasType(watcher.EventMovedTo):
		return "MOVED_TO"
	case event.HasType(watcher.EventAttrib):
		return "ATTRIB"
	default:
		return "UNKNOWN"
	}
}

func executeHook(hookCmd string, event *watcher.Event, eventType string) {
	isDirStr := "false"
	if event.IsDir {
		isDirStr = "true"
	}

	// Execute hook with event information as arguments
	cmd := exec.Command(hookCmd, eventType, event.Path, event.Name, isDirStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Hook command failed: %v", err)
	}
}
