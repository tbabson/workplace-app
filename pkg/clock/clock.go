// Package clock gives the application's notion of the current date.
//
// SQL must not use CURRENT_DATE: it follows the database session timezone,
// which connection poolers (e.g. Neon's PgBouncer endpoint) force to GMT.
// Pass clock.Today() as a query parameter instead.
package clock

import (
	"sync/atomic"
	"time"
)

var loc atomic.Pointer[time.Location]

// SetLocation sets the application timezone. Call once at startup.
func SetLocation(l *time.Location) { loc.Store(l) }

func location() *time.Location {
	if l := loc.Load(); l != nil {
		return l
	}
	return time.UTC
}

// Now returns the current time in the application timezone.
func Now() time.Time { return time.Now().In(location()) }

// Today returns today's date (YYYY-MM-DD) in the application timezone.
func Today() string { return DaysFromToday(0) }

// DaysFromToday returns the date n days from today (YYYY-MM-DD).
func DaysFromToday(n int) string { return Now().AddDate(0, 0, n).Format("2006-01-02") }

// Zone returns the IANA name of the application timezone, e.g. "Africa/Lagos".
func Zone() string { return location().String() }
