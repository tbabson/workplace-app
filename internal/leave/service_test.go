package leave

import (
	"testing"
	"time"
)

func TestWorkingDays(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	cases := []struct {
		start, end string
		want       int
	}{
		{"2026-11-23", "2026-11-23", 1}, // Monday
		{"2026-11-23", "2026-11-29", 5}, // Mon–Sun
		{"2026-11-28", "2026-11-29", 0}, // weekend only
		{"2026-11-27", "2026-11-30", 2}, // Fri–Mon
	}
	for _, c := range cases {
		if got := workingDays(d(c.start), d(c.end)); got != c.want {
			t.Errorf("workingDays(%s, %s) = %d, want %d", c.start, c.end, got, c.want)
		}
	}
}
