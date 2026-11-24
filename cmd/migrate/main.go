// Command migrate applies the SQL files in ./migrations to DATABASE_URL in
// filename order, recording each applied file in schema_migrations so it runs
// only once.
//
//	go run ./cmd/migrate                 # apply pending migrations
//	go run ./cmd/migrate -status         # list applied / pending files
//	go run ./cmd/migrate -baseline 010   # mark 001..010 as applied without running
//	                                     # them (for databases migrated by hand)
//
// Each statement runs on its own, like psql, so files that add an enum value
// and then use it (e.g. 003) work.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"workplace/pkg/config"

	"github.com/jackc/pgx/v5"
)

func main() {
	dir := flag.String("dir", "migrations", "directory containing .sql files")
	status := flag.Bool("status", false, "show which migrations are applied and exit")
	baseline := flag.String("baseline", "", "mark files whose name sorts <= this prefix as applied, without running them")
	flag.Parse()

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		filename   TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		log.Fatalf("create schema_migrations: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(*dir, "*.sql"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)

	applied := map[string]bool{}
	rows, err := conn.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		log.Fatalf("read schema_migrations: %v", err)
	}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			log.Fatal(err)
		}
		applied[f] = true
	}
	rows.Close()

	pending := 0
	for _, path := range files {
		name := filepath.Base(path)
		switch {
		case applied[name]:
			if *status {
				fmt.Println("applied  ", name)
			}
			continue
		case *status:
			fmt.Println("pending  ", name)
			pending++
			continue
		case *baseline != "" && name[:min(len(name), len(*baseline))] <= *baseline:
			record(ctx, conn, name)
			fmt.Println("baselined", name)
			continue
		}

		src, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		for i, stmt := range splitStatements(string(src)) {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				log.Fatalf("%s: statement %d failed: %v\n%s", name, i+1, err, stmt)
			}
		}
		record(ctx, conn, name)
		fmt.Println("applied  ", name)
	}

	if *status {
		fmt.Printf("%d pending\n", pending)
	}
}

func record(ctx context.Context, conn *pgx.Conn, name string) {
	if _, err := conn.Exec(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, name); err != nil {
		log.Fatalf("record %s: %v", name, err)
	}
}

// splitStatements splits SQL on top-level semicolons, ignoring semicolons inside
// comments, quoted strings and dollar-quoted bodies (DO $$ ... $$).
func splitStatements(sql string) []string {
	var (
		out    []string
		buf    strings.Builder
		dollar string // active dollar-quote tag, e.g. "$$"
	)
	flush := func() {
		if s := strings.TrimSpace(buf.String()); s != "" && !onlyComments(s) {
			out = append(out, s)
		}
		buf.Reset()
	}

	for i := 0; i < len(sql); i++ {
		c := sql[i]
		switch {
		case dollar != "":
			if strings.HasPrefix(sql[i:], dollar) {
				buf.WriteString(dollar)
				i += len(dollar) - 1
				dollar = ""
				continue
			}
		case c == '-' && strings.HasPrefix(sql[i:], "--"):
			end := strings.IndexByte(sql[i:], '\n')
			if end < 0 {
				end = len(sql) - i
			}
			buf.WriteString(sql[i : i+end])
			i += end - 1
			continue
		case c == '\'':
			// Copy the whole string literal; '' is an escaped quote.
			j := i + 1
			for j < len(sql) {
				if sql[j] == '\'' {
					if j+1 < len(sql) && sql[j+1] == '\'' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			buf.WriteString(sql[i:min(j+1, len(sql))])
			i = j
			continue
		case c == '$':
			if end := strings.IndexByte(sql[i+1:], '$'); end >= 0 {
				tag := sql[i : i+end+2]
				if isDollarTag(tag) {
					dollar = tag
					buf.WriteString(tag)
					i += len(tag) - 1
					continue
				}
			}
		case c == ';':
			flush()
			continue
		}
		buf.WriteByte(c)
	}
	flush()
	return out
}

func isDollarTag(tag string) bool {
	inner := tag[1 : len(tag)-1]
	for _, r := range inner {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return inner == "" || !(inner[0] >= '0' && inner[0] <= '9') // $1 is a parameter, not a tag
}

func onlyComments(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		if l := strings.TrimSpace(line); l != "" && !strings.HasPrefix(l, "--") {
			return false
		}
	}
	return true
}
