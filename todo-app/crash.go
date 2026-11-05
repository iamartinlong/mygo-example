package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// maxCrashes bounds recovery: a panic firing every frame would otherwise
// spin the app forever while filling the disk with logs.
const maxCrashes = 100

var crashCount atomic.Int64

// crashDrill injects one view panic when TODO_CRASH_TEST=1, to exercise the
// guard in automated verification.
var crashDrill = os.Getenv("TODO_CRASH_TEST") == "1"

// recoverFrame turns a view panic into a crash-log entry instead of process
// death. The aborted frame leaves the window blank — redraws are event-driven
// and the panic swallowed the input that would request one — so a redraw is
// pushed to the window once it is known.
func recoverFrame(wp **mygo.Window) {
	if r := recover(); r != nil {
		n := crashCount.Add(1)
		writeCrashLog(r)
		if n >= maxCrashes {
			fmt.Fprintln(os.Stderr, "view keeps panicking, giving up after", n, "crashes")
			os.Exit(1)
		}
		go func() {
			for i := 0; i < 500; i++ {
				if w := *wp; w != nil {
					w.Update(func() {})
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
}

func writeCrashLog(r any) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		dir = "."
	}
	f, err := os.OpenFile(filepath.Join(dir, "crash.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] panic: %v\n%s\n", time.Now().Format(time.RFC3339), r, debug.Stack())
}
