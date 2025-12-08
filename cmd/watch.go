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
and move operations.

Available modes:
  default        - Monitor all common events (create, modify, delete, move, etc.)
  write-complete - Monitor only write completion (CLOSE_WRITE, MOVED_TO)
                   Ideal for detecting when cp/rsync/scp operations finish.`,
	Example: `  # Watch for write completion only (recommended for artifact sync)
  truenas-artifact-inotify-hook watch /data/artifacts --mode=write-complete

  # Watch with a hook command triggered on write complete
  truenas-artifact-inotify-hook watch /data/artifacts --mode=write-complete \
    --hook=/usr/local/bin/on-artifact-ready.sh

  # Watch multiple directories
  truenas-artifact-inotify-hook watch /data/artifacts /data/uploads

  # Watch specific events only
  truenas-artifact-inotify-hook watch /data/artifacts --events=close_write

  # Non-recursive watch
  truenas-artifact-inotify-hook watch /data/artifacts --recursive=false`,
	Args: cobra.MinimumNArgs(1),
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Watch command flags
	watchCmd.Flags().StringP("mode", "m", "write-complete", "Watch mode: default, write-complete")
	watchCmd.Flags().BoolP("recursive", "r", true, "Watch directories recursively")
	watchCmd.Flags().StringSliceP("ignore", "i", []string{".git", "*.tmp", "*.swp", "*~"}, "Patterns to ignore")
	watchCmd.Flags().StringP("hook", "H", "", "Command to execute on file events")
	watchCmd.Flags().StringSliceP("events", "e", []string{}, "Filter specific events (create,modify,delete,move,close_write,attrib)")
	watchCmd.Flags().BoolP("dirs-only", "d", false, "Only report events for directories (ignore files)")
	watchCmd.Flags().BoolP("files-only", "f", false, "Only report events for files (ignore directories)")

	// Bind to viper
	viper.BindPFlag("watch.mode", watchCmd.Flags().Lookup("mode"))
	viper.BindPFlag("watch.recursive", watchCmd.Flags().Lookup("recursive"))
	viper.BindPFlag("watch.ignore", watchCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("watch.hook", watchCmd.Flags().Lookup("hook"))
	viper.BindPFlag("watch.events", watchCmd.Flags().Lookup("events"))
	viper.BindPFlag("watch.dirs-only", watchCmd.Flags().Lookup("dirs-only"))
	viper.BindPFlag("watch.files-only", watchCmd.Flags().Lookup("files-only"))
}

func runWatch(cmd *cobra.Command, args []string) error {
	paths := args
	mode := viper.GetString("watch.mode")
	recursive := viper.GetBool("watch.recursive")
	ignorePatterns := viper.GetStringSlice("watch.ignore")
	hookCommand := viper.GetString("watch.hook")
	eventFilter := viper.GetStringSlice("watch.events")
	dirsOnly := viper.GetBool("watch.dirs-only")
	filesOnly := viper.GetBool("watch.files-only")
	verbose := viper.GetBool("verbose")

	// Determine watch mask based on mode or explicit events
	var watchMask uint32
	switch mode {
	case "write-complete":
		watchMask = watcher.WriteCompleteWatchMask
		log.Printf("Mode: write-complete (CLOSE_WRITE, MOVED_TO)")
	case "default":
		watchMask = watcher.DefaultWatchMask
		log.Printf("Mode: default (all events)")
	default:
		watchMask = watcher.WriteCompleteWatchMask
		log.Printf("Mode: write-complete (CLOSE_WRITE, MOVED_TO)")
	}

	// Override with explicit events if specified
	if len(eventFilter) > 0 {
		watchMask = buildEventMask(eventFilter)
		log.Printf("Custom events: %v", eventFilter)
	}

	// Create event handler
	eventHandler := createEventHandler(hookCommand, eventFilter, verbose, dirsOnly, filesOnly)

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

func createEventHandler(hookCommand string, eventFilter []string, verbose, dirsOnly, filesOnly bool) watcher.EventHandler {
	return func(event *watcher.Event) {
		// Filter by type (directory/file)
		if dirsOnly && !event.IsDir {
			return
		}
		if filesOnly && event.IsDir {
			return
		}

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
