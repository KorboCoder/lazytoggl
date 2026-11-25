package ui

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
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
		// TZ naming a zone the system lacks silently falls back to UTC.
		if got := time.Local.String(); got != tz {
			t.Fatalf("child runs in %q, want %q", got, tz)
		}
		return true
	}
	if _, err := time.LoadLocation(tz); err != nil {
		t.Skipf("time zone %s unavailable: %v", tz, err)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^"+regexp.QuoteMeta(t.Name())+"$", "-test.count=1", "-test.v")
	cmd.Env = append(os.Environ(), "TZ="+tz, "LAZYTOGGL_TZ_CHILD="+tz)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("in %s: %v\n%s", tz, err, out)
	}
	// A child that matched no test would pass without running anything.
	if !strings.Contains(string(out), "--- PASS: "+t.Name()) {
		t.Fatalf("in %s: the test didn't run\n%s", tz, out)
	}
	return false
}
