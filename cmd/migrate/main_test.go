package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	sql := `-- leading comment
CREATE TABLE a (x TEXT DEFAULT 'a;b');
DO $$ BEGIN
    CREATE TYPE t AS ENUM ('x');
EXCEPTION WHEN duplicate_object THEN null;
END $$;
-- comment with ; semicolon
ALTER TYPE t ADD VALUE IF NOT EXISTS 'y';
SELECT 'it''s; fine';`

	got := splitStatements(sql)
	if len(got) != 4 {
		t.Fatalf("got %d statements, want 4:\n%s", len(got), strings.Join(got, "\n---\n"))
	}
	if !strings.Contains(got[1], "END $$") || !strings.Contains(got[1], "duplicate_object THEN null;") {
		t.Errorf("DO block split incorrectly: %q", got[1])
	}
	if !strings.HasSuffix(got[3], "'it''s; fine'") {
		t.Errorf("escaped quote handled incorrectly: %q", got[3])
	}
}

// Every real migration should split into statements that each look complete.
func TestSplitRealMigrations(t *testing.T) {
	files, _ := filepath.Glob("../../migrations/*.sql")
	if len(files) == 0 {
		t.Skip("no migrations found")
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, stmt := range splitStatements(string(src)) {
			if strings.Count(stmt, "$$")%2 != 0 {
				t.Errorf("%s: unbalanced $$ in statement:\n%s", filepath.Base(f), stmt)
			}
		}
	}
}
