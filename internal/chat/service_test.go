package chat

import "testing"

func TestCanManageMembers(t *testing.T) {
	general := &Room{Type: RoomGeneral, CreatedBy: "creator"}
	dm := &Room{Type: RoomDirect, CreatedBy: "creator"}
	proj := &Room{Type: RoomProject, CreatedBy: "creator"}

	cases := []struct {
		name         string
		room         *Room
		caller, role string
		want         bool
	}{
		{"creator manages own room", general, "creator", "staff", true},
		{"stranger cannot manage room", general, "other", "staff", false},
		{"dept head cannot manage others' room", general, "other", "dept_head", false},
		{"admin manages any room", general, "other", "super_admin", true},
		{"nobody edits a DM", dm, "creator", "staff", false},
		{"not even admin edits a DM", dm, "other", "super_admin", false},
		{"creator cannot edit project room", proj, "creator", "staff", false},
		{"admin can edit project room", proj, "other", "super_admin", true},
	}
	for _, c := range cases {
		if got := canManageMembers(c.room, c.caller, c.role); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestTruncateKeepsRunesWhole(t *testing.T) {
	if got := truncate("héllo wörld", 4); got != "héll…" {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate = %q", got)
	}
}
