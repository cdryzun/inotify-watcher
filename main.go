// TrueNAS Artifact Inotify Hook
// A file system monitoring tool using Linux inotify via golang.org/x/sys/unix.
package main

import (
	"gitlab.cpinnov.run/devops/truenas-artifact-inotify-hook/cmd"
)

func main() {
	cmd.Execute()
}
