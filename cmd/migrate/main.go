package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	target := os.Args[2]

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("set dialect", "err", err)
		os.Exit(1)
	}

	switch {
	case target == "public":
		if err := run(db, "public", "public", command); err != nil {
			slog.Error("migration failed", "target", target, "err", err)
			os.Exit(1)
		}

	case strings.HasPrefix(target, "tenant:"):
		slug := strings.TrimPrefix(target, "tenant:")
		if slug == "*" {
			tenants, err := allTenantSlugs(db)
			if err != nil {
				slog.Error("list tenants", "err", err)
				os.Exit(1)
			}
			for _, s := range tenants {
				if err := run(db, "tenant_"+s, "tenant", command); err != nil {
					slog.Error("migration failed", "tenant", s, "err", err)
					os.Exit(1)
				}
			}
		} else {
			if err := run(db, "tenant_"+slug, "tenant", command); err != nil {
				slog.Error("migration failed", "tenant", slug, "err", err)
				os.Exit(1)
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown target: %s\n\n", target)
		printUsage()
		os.Exit(1)
	}
}

// run creates the schema if needed, sets search_path, then runs the goose command
// against the SQL files in the given directory of the embedded FS.
func run(db *sql.DB, schema, dir, command string) error {
	quoted := `"` + schema + `"`
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + quoted); err != nil {
		return fmt.Errorf("create schema %s: %w", schema, err)
	}
	if _, err := db.Exec("SET search_path TO " + quoted); err != nil {
		return fmt.Errorf("set search_path to %s: %w", schema, err)
	}

	switch command {
	case "up":
		return goose.Up(db, dir)
	case "up-one":
		return goose.UpByOne(db, dir)
	case "down":
		return goose.Down(db, dir)
	case "reset":
		return goose.Reset(db, dir)
	case "status":
		return goose.Status(db, dir)
	case "version":
		return goose.Version(db, dir)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func allTenantSlugs(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT slug FROM public.tenants ORDER BY slug")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <command> <target>")
	fmt.Fprintln(os.Stderr, "  commands : up | up-one | down | reset | status | version")
	fmt.Fprintln(os.Stderr, "  targets  : public | tenant:<slug> | tenant:*")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "examples:")
	fmt.Fprintln(os.Stderr, "  migrate up public")
	fmt.Fprintln(os.Stderr, "  migrate up tenant:sunrise-dairy")
	fmt.Fprintln(os.Stderr, "  migrate status tenant:*")
}
