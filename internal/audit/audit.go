package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WritePublicEvent writes to public.events (auth, tenant lifecycle events).
// Call within a transaction that has search_path = public.
func WritePublicEvent(ctx context.Context, tx pgx.Tx, tenantID, actorID *string, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("audit: marshal: %w", err)
	}
	_, err = tx.Exec(ctx,
		"INSERT INTO events (tenant_id, actor_id, event_type, payload) VALUES ($1, $2, $3, $4)",
		tenantID, actorID, eventType, data,
	)
	if err != nil {
		return fmt.Errorf("audit: write public event %s: %w", eventType, err)
	}
	return nil
}

// WriteTenantEvent writes to the tenant schema events table (business events).
// Call within a transaction that has the correct tenant search_path set.
func WriteTenantEvent(ctx context.Context, tx pgx.Tx, actorID *string, eventType, entityType string, entityID *string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("audit: marshal: %w", err)
	}
	_, err = tx.Exec(ctx,
		"INSERT INTO events (actor_id, event_type, entity_type, entity_id, payload) VALUES ($1, $2, $3, $4, $5)",
		actorID, eventType, entityType, entityID, data,
	)
	if err != nil {
		return fmt.Errorf("audit: write tenant event %s: %w", eventType, err)
	}
	return nil
}
