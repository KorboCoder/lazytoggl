package ui

import (
	"os"
	"os/exec"
	"testing"
)

// inTimeZone runs the calling test in a child process whose local zone is tz
// (set through TZ, so time.Local is fixed at startup). It returns true in the
// child, where the test body runs, and false in the parent, which only waits
// for the child.
//
// Tests must not assign time.Local themselves: other goroutines (teatest's
// event loop, its leftover tick and blink timers) read it concurrently, and
// the race detector rightly flags the write.
func inTimeZone(t *testing.T, tz string) bool {
	t.Helper()
	if os.Getenv("LAZYTOGGL_TZ_CHILD") == tz {
		return true
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^"+t.Name()+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), "TZ="+tz, "LAZYTOGGL_TZ_CHILD="+tz)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("in %s: %v\n%s", tz, err, out)
	}
	return false
}
