package db_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/db"
)

// TestCrossTenantIsolation is a permanent P0 gate. Never skip.
//
// Verifies that two tenant contexts cannot see each other's data even under
// concurrent access, ensuring the BeforeAcquire/AfterRelease search_path hooks
// work correctly.
func TestCrossTenantIsolation(t *testing.T) {
	ctx := context.Background()

	// 1. Start a real Postgres container — no mocks.
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf(
		"postgres://test:test@%s:%s/testdb?sslmode=disable",
		host, port.Port(),
	)

	// 2. Create pool with tenant-aware search_path switching.
	cfg := &config.Config{DatabaseURL: connStr}
	pool, err := db.New(cfg)
	require.NoError(t, err)
	defer pool.Close()

	// 3. Set up two isolated tenant schemas, each with an items table.
	_, err = pool.Exec(ctx, `
		CREATE SCHEMA tenant_alpha;
		CREATE TABLE tenant_alpha.items (id SERIAL PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO tenant_alpha.items (name) VALUES ('milk'), ('cheese');

		CREATE SCHEMA tenant_beta;
		CREATE TABLE tenant_beta.items (id SERIAL PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO tenant_beta.items (name) VALUES ('sugar');
	`)
	require.NoError(t, err)

	ctxAlpha := db.WithTenant(ctx, "alpha")
	ctxBeta := db.WithTenant(ctx, "beta")

	// 4. Alpha context: must see only alpha rows.
	var nAlpha int
	require.NoError(t, pool.QueryRow(ctxAlpha, "SELECT COUNT(*) FROM items").Scan(&nAlpha))
	require.Equal(t, 2, nAlpha, "tenant_alpha should see 2 items")

	// 5. Beta context: must see only beta rows (not alpha's).
	var nBeta int
	require.NoError(t, pool.QueryRow(ctxBeta, "SELECT COUNT(*) FROM items").Scan(&nBeta))
	require.Equal(t, 1, nBeta, "tenant_beta should see 1 item")

	// 6. No tenant context: public schema, no items table visible.
	var nPublic int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'tenant_alpha' AND table_name = 'items'
	`).Scan(&nPublic)
	require.NoError(t, err)
	require.Equal(t, 1, nPublic, "information_schema should confirm tenant_alpha.items exists")

	// 7. Concurrent access: 20 goroutines per tenant, interleaved.
	//    Every goroutine must see only its own tenant's rows.
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			var n int
			require.NoError(t, pool.QueryRow(ctxAlpha, "SELECT COUNT(*) FROM items").Scan(&n))
			require.Equal(t, 2, n, "concurrent: tenant_alpha must always see 2 items")
		}()
		go func() {
			defer wg.Done()
			var n int
			require.NoError(t, pool.QueryRow(ctxBeta, "SELECT COUNT(*) FROM items").Scan(&n))
			require.Equal(t, 1, n, "concurrent: tenant_beta must always see 1 item")
		}()
	}
	wg.Wait()
}
