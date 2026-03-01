// TrueNAS Artifact Inotify Hook
// A file system monitoring tool using Linux inotify via golang.org/x/sys/unix.
package main

import (
	"github.com/cdryzun/inotify-watcher/cmd"
)

func main() {
	cmd.Execute()
}
