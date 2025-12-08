package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
                   Ideal for detecting when cp/rsync/scp operations finish.

Hook Debounce (--debounce):
  When a hook command is configured, the debounce mechanism prevents excessive
  hook executions during rapid file operations (e.g., rsync, bulk file copies).

  How it works:
    - Events within the debounce window are aggregated
    - Hook executes only once after the window expires with no new events
    - Uses the most recent event for hook parameters

  Recommended values:
    500ms  - Default, good for most use cases
    1000ms - Large file transfers or slow hooks
    2000ms - Very large batch operations
    0      - Disable debounce (execute hook for every event)`,
	Example: `  # Watch for write completion only (recommended for artifact sync)
  truenas-artifact-inotify-hook watch /data/artifacts --mode=write-complete

  # Watch with a hook command triggered on write complete
  truenas-artifact-inotify-hook watch /data/artifacts --mode=write-complete \
    --hook=/usr/local/bin/on-artifact-ready.sh

  # Watch multiple directories with 1 second debounce
  truenas-artifact-inotify-hook watch /data/prod /data/test /data/pre \
    --hook=/usr/local/bin/sync.sh --debounce=1000

  # Disable debounce for immediate hook execution on each event
  truenas-artifact-inotify-hook watch /data/artifacts \
    --hook=/usr/local/bin/notify.sh --debounce=0

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
	watchCmd.Flags().Int("debounce", 500, "Hook debounce window in ms; aggregates events and executes hook once after window expires (0=disable, execute on every event)")

	// Bind to viper
	viper.BindPFlag("watch.mode", watchCmd.Flags().Lookup("mode"))
	viper.BindPFlag("watch.recursive", watchCmd.Flags().Lookup("recursive"))
	viper.BindPFlag("watch.ignore", watchCmd.Flags().Lookup("ignore"))
	viper.BindPFlag("watch.hook", watchCmd.Flags().Lookup("hook"))
	viper.BindPFlag("watch.events", watchCmd.Flags().Lookup("events"))
	viper.BindPFlag("watch.dirs-only", watchCmd.Flags().Lookup("dirs-only"))
	viper.BindPFlag("watch.files-only", watchCmd.Flags().Lookup("files-only"))
	viper.BindPFlag("watch.debounce", watchCmd.Flags().Lookup("debounce"))
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
	debounceMs := viper.GetInt("watch.debounce")
	verbose := viper.GetBool("verbose")

	if dirsOnly && filesOnly {
		return fmt.Errorf("cannot use --dirs-only and --files-only together")
	}

	// Determine watch mask based on mode or explicit events
	var watchMask uint32
	var isWriteCompleteMode bool
	switch mode {
	case "write-complete":
		watchMask = watcher.WriteCompleteWatchMask
		isWriteCompleteMode = true
		log.Printf("Mode: write-complete (CLOSE_WRITE, MOVED_TO)")
	case "default":
		watchMask = watcher.DefaultWatchMask
		log.Printf("Mode: default (all events)")
	default:
		watchMask = watcher.WriteCompleteWatchMask
		isWriteCompleteMode = true
		log.Printf("Mode: write-complete (CLOSE_WRITE, MOVED_TO)")
	}

	// Override with explicit events if specified
	if len(eventFilter) > 0 {
		watchMask = buildEventMask(eventFilter)
		log.Printf("Custom events: %v", eventFilter)
	}

	// Create event handler
	debounceTime := time.Duration(debounceMs) * time.Millisecond
	eventHandler := createEventHandler(hookCommand, eventFilter, verbose, dirsOnly, filesOnly, debounceTime, isWriteCompleteMode)

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
	var stopOnce sync.Once
	stopWatcher := func() {
		stopOnce.Do(func() {
			if err := w.Stop(); err != nil {
				log.Printf("Error stopping watcher: %v", err)
			}
		})
	}
	defer stopWatcher()

	// Add paths to watch concurrently for better performance
	log.Printf("Adding %d paths to watch...", len(paths))

	var wg sync.WaitGroup
	errChanAdd := make(chan error, len(paths))
	pathResults := make(chan string, len(paths))

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := w.Add(p); err != nil {
				errChanAdd <- fmt.Errorf("failed to watch path %s: %w", p, err)
				return
			}
			pathResults <- p
		}(path)
	}

	// Wait for all paths to be added
	go func() {
		wg.Wait()
		close(errChanAdd)
		close(pathResults)
	}()

	// Collect results
	var addErrors []error
	var watchedPaths []string

	for {
		select {
		case err, ok := <-errChanAdd:
			if ok && err != nil {
				addErrors = append(addErrors, err)
			}
		case p, ok := <-pathResults:
			if ok {
				watchedPaths = append(watchedPaths, p)
			}
		}
		// Break when both channels are closed
		if len(addErrors)+len(watchedPaths) >= len(paths) {
			break
		}
	}

	// Report results
	for _, p := range watchedPaths {
		log.Printf("Watching: %s", p)
	}

	if len(addErrors) > 0 {
		for _, err := range addErrors {
			log.Printf("Warning: %v", err)
		}
		if len(watchedPaths) == 0 {
			return fmt.Errorf("failed to watch any paths")
		}
	}

	// Show total watch count
	totalWatches := len(w.WatchedPaths())
	log.Printf("Total directories being watched: %d", totalWatches)

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

	stopWatcher()
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
		case "move":
			mask |= uint32(watcher.EventMovedFrom)
			mask |= uint32(watcher.EventMovedTo)
		case "moved_from":
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

// hookDebouncer provides debounce control for hook execution
type hookDebouncer struct {
	mu           sync.Mutex
	timer        *time.Timer
	debounceTime time.Duration
	pending      []*watcher.Event
	hookCmd      string
	running      bool
}

func newHookDebouncer(hookCmd string, debounceTime time.Duration) *hookDebouncer {
	return &hookDebouncer{
		hookCmd:      hookCmd,
		debounceTime: debounceTime,
		pending:      make([]*watcher.Event, 0),
	}
}

func (d *hookDebouncer) trigger(event *watcher.Event, eventType string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.pending = append(d.pending, event)

	// Reset timer on each event
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.debounceTime, func() {
		d.execute()
	})
}

func (d *hookDebouncer) execute() {
	d.mu.Lock()
	if d.running || len(d.pending) == 0 {
		d.mu.Unlock()
		return
	}
	d.running = true
	events := d.pending
	d.pending = make([]*watcher.Event, 0)
	d.mu.Unlock()

	// Log aggregated events
	log.Printf("Executing hook for %d aggregated events", len(events))

	// Use the last event for hook execution (most recent)
	lastEvent := events[len(events)-1]
	eventType := getEventTypeString(lastEvent)

	executeHook(d.hookCmd, lastEvent, eventType)

	d.mu.Lock()
	d.running = false
	d.mu.Unlock()
}

func createEventHandler(hookCommand string, eventFilter []string, verbose, dirsOnly, filesOnly bool, debounceTime time.Duration, isWriteCompleteMode bool) watcher.EventHandler {
	// Create debouncer for hook execution
	var debouncer *hookDebouncer
	if hookCommand != "" && debounceTime > 0 {
		debouncer = newHookDebouncer(hookCommand, debounceTime)
		log.Printf("Hook debounce: %v", debounceTime)
	}

	return func(event *watcher.Event) {
		// In write-complete mode, filter out CREATE events (they're only used for recursive dir watching)
		// Only report CLOSE_WRITE and MOVED_TO events
		if isWriteCompleteMode && event.HasType(watcher.EventCreate) && !event.HasType(watcher.EventCloseWrite) && !event.HasType(watcher.EventMovedTo) {
			return
		}

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
			if debouncer != nil {
				debouncer.trigger(event, eventType)
			} else {
				// No debounce, execute directly in goroutine
				go executeHook(hookCommand, event, eventType)
			}
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
		case "move":
			if event.HasType(watcher.EventMovedFrom) || event.HasType(watcher.EventMovedTo) {
				return true
			}
		case "moved_from":
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
	// Collect all matching event types (inotify events can have multiple flags)
	var types []string

	if event.HasType(watcher.EventCreate) {
		types = append(types, "CREATE")
	}
	if event.HasType(watcher.EventCloseWrite) {
		types = append(types, "CLOSE_WRITE")
	}
	if event.HasType(watcher.EventModify) {
		types = append(types, "MODIFY")
	}
	if event.HasType(watcher.EventDelete) {
		types = append(types, "DELETE")
	}
	if event.HasType(watcher.EventDeleteSelf) {
		types = append(types, "DELETE_SELF")
	}
	if event.HasType(watcher.EventMovedFrom) {
		types = append(types, "MOVED_FROM")
	}
	if event.HasType(watcher.EventMovedTo) {
		types = append(types, "MOVED_TO")
	}
	if event.HasType(watcher.EventAttrib) {
		types = append(types, "ATTRIB")
	}
	if event.HasType(watcher.EventIgnored) {
		types = append(types, "IGNORED")
	}
	if event.HasType(watcher.EventQueueOverflow) {
		types = append(types, "Q_OVERFLOW")
	}

	if len(types) == 0 {
		return fmt.Sprintf("UNKNOWN(0x%x)", event.Mask)
	}
	return strings.Join(types, "|")
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
