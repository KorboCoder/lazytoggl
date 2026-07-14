package theme

import (
	"testing"
	"time"
)

func TestFormatClock(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00:00"},
		{61 * time.Second, "00:01:01"},
		{3661 * time.Second, "01:01:01"},
		{26*time.Hour + 3*time.Minute, "26:03:00"},
	}
	for _, c := range cases {
		if got := FormatClock(c.d); got != c.want {
			t.Errorf("FormatClock(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFormatTotal(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{37 * time.Minute, "37m"},
		{3*time.Hour + 45*time.Minute, "3h 45m"},
		{2 * time.Hour, "2h 00m"},
	}
	for _, c := range cases {
		if got := FormatTotal(c.d); got != c.want {
			t.Errorf("FormatTotal(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		w    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"héllo wörld", 6, "héllo…"}, // rune-aware, not byte-aware
	}
	for _, c := range cases {
		if got := Truncate(c.s, c.w); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}
}
