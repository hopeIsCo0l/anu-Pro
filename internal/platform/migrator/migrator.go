package migrator

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hopeIsCo0l/anu-pro/migrations"
)

// ProvisionTenant creates the tenant_<slug> schema and applies all tenant
// migrations to it. Safe to call multiple times — CREATE SCHEMA IF NOT EXISTS
// and goose are both idempotent.
func ProvisionTenant(databaseURL, slug string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	schema := "tenant_" + slug
	quoted := `"` + schema + `"`
	if _, err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + quoted); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	if _, err := db.Exec("SET search_path TO " + quoted); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}

	return goose.Up(db, "tenant")
}
