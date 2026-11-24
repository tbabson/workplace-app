package clock

import (
	"testing"
	"time"
)

func TestTodayUsesConfiguredZone(t *testing.T) {
	// Pick zones far apart so at least one differs from UTC's date at any moment.
	for _, name := range []string{"Pacific/Kiritimati", "Pacific/Pago_Pago"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Skipf("tzdata missing: %v", err)
		}
		SetLocation(loc)
		if want := time.Now().In(loc).Format("2006-01-02"); Today() != want {
			t.Errorf("%s: Today() = %s, want %s", name, Today(), want)
		}
		if Zone() != name {
			t.Errorf("Zone() = %s, want %s", Zone(), name)
		}
	}
}
