package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ── Items ─────────────────────────────────────────────────────────────────────

func (s *Service) CreateItem(ctx context.Context, req CreateItemRequest) (*Item, error) {
	if req.SKU == "" || req.Name == "" || req.Unit == "" {
		return nil, fmt.Errorf("sku, name, and unit are required")
	}
	if req.Attributes == nil {
		req.Attributes = []byte("{}")
	}

	var minQtyStr *string
	if req.MinStockQty != nil {
		str := req.MinStockQty.String()
		minQtyStr = &str
	}

	item := &Item{}
	var rawMinQty *string
	var attrs []byte

	err := s.pool.QueryRow(ctx, `
		INSERT INTO items (sku, name, description, unit, category, min_stock_qty, attributes)
		VALUES ($1, $2, $3, $4, $5, $6::NUMERIC, $7)
		RETURNING id, sku, name, description, unit, category,
		          min_stock_qty::TEXT, attributes, is_active, created_at, updated_at
	`, req.SKU, req.Name, req.Description, req.Unit, req.Category, minQtyStr, req.Attributes,
	).Scan(
		&item.ID, &item.SKU, &item.Name, &item.Description, &item.Unit, &item.Category,
		&rawMinQty, &attrs, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateSKU
		}
		return nil, fmt.Errorf("create item: %w", err)
	}

	item.MinStockQty = parseNullDecimal(rawMinQty)
	item.Attributes = attrs
	return item, nil
}

func (s *Service) ListItems(ctx context.Context, search, category *string, activeOnly bool, limit, offset int) ([]Item, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, sku, name, description, unit, category,
		       min_stock_qty::TEXT, attributes, is_active, created_at, updated_at
		FROM items
		WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
		  AND ($2::TEXT IS NULL OR category = $2)
		  AND ($3::TEXT IS NULL OR to_tsvector('english', name) @@ plainto_tsquery('english', $3)
		                        OR sku ILIKE '%' || $3 || '%')
		ORDER BY name
		LIMIT $4 OFFSET $5
	`, activeOnly, category, search, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	return scanItems(rows)
}

func (s *Service) GetItem(ctx context.Context, id string) (*Item, error) {
	item := &Item{}
	var rawMinQty *string
	var attrs []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, sku, name, description, unit, category,
		       min_stock_qty::TEXT, attributes, is_active, created_at, updated_at
		FROM items WHERE id = $1
	`, id).Scan(
		&item.ID, &item.SKU, &item.Name, &item.Description, &item.Unit, &item.Category,
		&rawMinQty, &attrs, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}

	item.MinStockQty = parseNullDecimal(rawMinQty)
	item.Attributes = attrs
	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, id string, req UpdateItemRequest) (*Item, error) {
	var minQtyStr *string
	if req.MinStockQty != nil {
		str := req.MinStockQty.String()
		minQtyStr = &str
	}
	if req.Attributes == nil {
		req.Attributes = []byte("{}")
	}

	item := &Item{}
	var rawMinQty *string
	var attrs []byte

	err := s.pool.QueryRow(ctx, `
		UPDATE items SET
			name         = COALESCE($2, name),
			description  = COALESCE($3, description),
			unit         = COALESCE($4, unit),
			category     = COALESCE($5, category),
			min_stock_qty = COALESCE($6::NUMERIC, min_stock_qty),
			attributes   = COALESCE($7, attributes),
			updated_at   = now()
		WHERE id = $1
		RETURNING id, sku, name, description, unit, category,
		          min_stock_qty::TEXT, attributes, is_active, created_at, updated_at
	`, id, req.Name, req.Description, req.Unit, req.Category, minQtyStr, req.Attributes,
	).Scan(
		&item.ID, &item.SKU, &item.Name, &item.Description, &item.Unit, &item.Category,
		&rawMinQty, &attrs, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("update item: %w", err)
	}

	item.MinStockQty = parseNullDecimal(rawMinQty)
	item.Attributes = attrs
	return item, nil
}

func (s *Service) DeactivateItem(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		"UPDATE items SET is_active = FALSE, updated_at = now() WHERE id = $1 AND is_active = TRUE",
		id,
	)
	if err != nil {
		return fmt.Errorf("deactivate item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

// ── Locations ─────────────────────────────────────────────────────────────────

func (s *Service) CreateLocation(ctx context.Context, req CreateLocationRequest) (*Location, error) {
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("code and name are required")
	}
	if req.Type == "" {
		req.Type = "zone"
	}

	loc := &Location{}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO locations (code, name, parent_id, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, name, parent_id, type, is_active
	`, req.Code, req.Name, req.ParentID, req.Type,
	).Scan(&loc.ID, &loc.Code, &loc.Name, &loc.ParentID, &loc.Type, &loc.IsActive)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("location code already exists")
		}
		return nil, fmt.Errorf("create location: %w", err)
	}
	return loc, nil
}

func (s *Service) ListLocations(ctx context.Context) ([]Location, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, name, parent_id, type, is_active
		FROM locations
		WHERE is_active = TRUE
		ORDER BY code
	`)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()

	var locs []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.ParentID, &l.Type, &l.IsActive); err != nil {
			return nil, err
		}
		locs = append(locs, l)
	}
	return locs, rows.Err()
}

func (s *Service) GetLocation(ctx context.Context, id string) (*Location, error) {
	loc := &Location{}
	err := s.pool.QueryRow(ctx,
		"SELECT id, code, name, parent_id, type, is_active FROM locations WHERE id = $1",
		id,
	).Scan(&loc.ID, &loc.Code, &loc.Name, &loc.ParentID, &loc.Type, &loc.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("get location: %w", err)
	}
	return loc, nil
}

// ── Lots ──────────────────────────────────────────────────────────────────────

func (s *Service) CreateLot(ctx context.Context, req CreateLotRequest) (*Lot, error) {
	if req.ItemID == "" || req.LotNumber == "" {
		return nil, fmt.Errorf("item_id and lot_number are required")
	}
	if req.ReceivedAt == nil {
		now := time.Now()
		req.ReceivedAt = &now
	}
	if req.Attributes == nil {
		req.Attributes = []byte("{}")
	}

	lot := &Lot{}
	var attrs []byte
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lots (item_id, lot_number, supplier_ref, received_at, expires_at, attributes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, item_id, lot_number, supplier_ref, received_at, expires_at, status, attributes
	`, req.ItemID, req.LotNumber, req.SupplierRef, req.ReceivedAt, req.ExpiresAt, req.Attributes,
	).Scan(
		&lot.ID, &lot.ItemID, &lot.LotNumber, &lot.SupplierRef,
		&lot.ReceivedAt, &lot.ExpiresAt, &lot.Status, &attrs,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateLot
		}
		return nil, fmt.Errorf("create lot: %w", err)
	}
	lot.Attributes = attrs
	return lot, nil
}

func (s *Service) ListLots(ctx context.Context, itemID *string, limit, offset int) ([]Lot, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, item_id, lot_number, supplier_ref, received_at, expires_at, status, attributes
		FROM lots
		WHERE ($1::UUID IS NULL OR item_id = $1)
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3
	`, itemID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()

	var lots []Lot
	for rows.Next() {
		var l Lot
		var attrs []byte
		if err := rows.Scan(&l.ID, &l.ItemID, &l.LotNumber, &l.SupplierRef,
			&l.ReceivedAt, &l.ExpiresAt, &l.Status, &attrs); err != nil {
			return nil, err
		}
		l.Attributes = attrs
		lots = append(lots, l)
	}
	return lots, rows.Err()
}

func (s *Service) GetLot(ctx context.Context, id string) (*Lot, error) {
	lot := &Lot{}
	var attrs []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, item_id, lot_number, supplier_ref, received_at, expires_at, status, attributes
		FROM lots WHERE id = $1
	`, id).Scan(
		&lot.ID, &lot.ItemID, &lot.LotNumber, &lot.SupplierRef,
		&lot.ReceivedAt, &lot.ExpiresAt, &lot.Status, &attrs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLotNotFound
		}
		return nil, fmt.Errorf("get lot: %w", err)
	}
	lot.Attributes = attrs
	return lot, nil
}

// ── Stock movements ───────────────────────────────────────────────────────────

func (s *Service) RecordMovement(ctx context.Context, req RecordMovementRequest, actorID string) (*StockMovement, error) {
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}
	if req.Quantity.IsZero() {
		return nil, fmt.Errorf("quantity must not be zero")
	}
	switch req.MovementType {
	case MovementReceipt, MovementAdjustment:
	default:
		return nil, fmt.Errorf("movement_type must be receipt or adjustment (use /stock/transfers for transfers)")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Verify lot and location exist
	var lotExists, locExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM lots WHERE id = $1)", req.LotID).Scan(&lotExists); err != nil {
		return nil, fmt.Errorf("check lot: %w", err)
	}
	if !lotExists {
		return nil, ErrLotNotFound
	}
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1 AND is_active)", req.LocationID).Scan(&locExists); err != nil {
		return nil, fmt.Errorf("check location: %w", err)
	}
	if !locExists {
		return nil, ErrLocationNotFound
	}

	var unitCostStr *string
	if req.UnitCost != nil {
		str := req.UnitCost.String()
		unitCostStr = &str
	}

	mov, err := insertMovement(ctx, tx, req.LotID, req.LocationID, req.Quantity.String(),
		req.MovementType, req.ReferenceID, req.IdempotencyKey, unitCostStr, req.Notes, actorID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateMovement
		}
		return nil, fmt.Errorf("insert movement: %w", err)
	}

	// Audit event
	if _, err := tx.Exec(ctx,
		"INSERT INTO events (actor_id, event_type, entity_type, entity_id, payload) VALUES ($1, $2, 'stock_movement', $3, '{}')",
		actorID, "inventory.movement."+req.MovementType, &mov.ID,
	); err != nil {
		return nil, fmt.Errorf("write event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return mov, nil
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest, actorID string) ([]StockMovement, error) {
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}
	if req.Quantity.IsZero() || req.Quantity.IsNegative() {
		return nil, fmt.Errorf("quantity must be positive for transfers")
	}
	if req.SourceLocationID == req.DestLocationID {
		return nil, fmt.Errorf("source and destination locations must differ")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	refID := uuid.New().String()

	out, err := insertMovement(ctx, tx, req.LotID, req.SourceLocationID,
		req.Quantity.Neg().String(), MovementTransfer, &refID,
		req.IdempotencyKey+":out", nil, req.Notes, actorID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateMovement
		}
		return nil, fmt.Errorf("insert outgoing movement: %w", err)
	}

	in, err := insertMovement(ctx, tx, req.LotID, req.DestLocationID,
		req.Quantity.String(), MovementTransfer, &refID,
		req.IdempotencyKey+":in", nil, req.Notes, actorID)
	if err != nil {
		return nil, fmt.Errorf("insert incoming movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transfer: %w", err)
	}
	return []StockMovement{*out, *in}, nil
}

func (s *Service) GetCurrentStock(ctx context.Context, itemID *string) ([]StockLevel, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cs.lot_id, cs.location_id, cs.quantity::TEXT,
		       i.id, i.name, i.sku, i.unit,
		       l.lot_number,
		       loc.code, loc.name
		FROM current_stock cs
		JOIN lots l    ON l.id = cs.lot_id
		JOIN items i   ON i.id = l.item_id
		JOIN locations loc ON loc.id = cs.location_id
		WHERE ($1::UUID IS NULL OR i.id = $1)
		ORDER BY i.name, l.lot_number, loc.code
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("get current stock: %w", err)
	}
	defer rows.Close()

	var levels []StockLevel
	for rows.Next() {
		var sl StockLevel
		var qtyStr string
		if err := rows.Scan(&sl.LotID, &sl.LocationID, &qtyStr,
			&sl.ItemID, &sl.ItemName, &sl.ItemSKU, &sl.Unit,
			&sl.LotNumber, &sl.LocationCode, &sl.LocationName); err != nil {
			return nil, err
		}
		sl.Quantity = parseDecimal(qtyStr)
		levels = append(levels, sl)
	}
	return levels, rows.Err()
}

func (s *Service) ListMovements(ctx context.Context, lotID *string, limit, offset int) ([]StockMovement, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, lot_id, location_id, quantity::TEXT, movement_type,
		       reference_id, unit_cost::TEXT, notes, actor_id, created_at
		FROM stock_movements
		WHERE ($1::UUID IS NULL OR lot_id = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, lotID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var movs []StockMovement
	for rows.Next() {
		var m StockMovement
		var qtyStr, unitCostStr *string
		var qtyRaw string
		if err := rows.Scan(&m.ID, &m.LotID, &m.LocationID, &qtyRaw, &m.MovementType,
			&m.ReferenceID, &unitCostStr, &m.Notes, &m.ActorID, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Quantity = parseDecimal(qtyRaw)
		m.UnitCost = parseNullDecimal(qtyStr)
		_ = unitCostStr // already scanned above — fix below
		m.UnitCost = parseNullDecimal(unitCostStr)
		movs = append(movs, m)
	}
	return movs, rows.Err()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func insertMovement(ctx context.Context, tx pgx.Tx, lotID, locationID, qty, movType string,
	refID *string, idempotencyKey string, unitCost, notes *string, actorID string) (*StockMovement, error) {

	mov := &StockMovement{}
	var qtyStr, unitCostStr *string

	err := tx.QueryRow(ctx, `
		INSERT INTO stock_movements
		       (lot_id, location_id, quantity, movement_type, reference_id, idempotency_key, unit_cost, notes, actor_id)
		VALUES ($1, $2, $3::NUMERIC, $4, $5, $6, $7::NUMERIC, $8, $9)
		RETURNING id, lot_id, location_id, quantity::TEXT, movement_type,
		          reference_id, unit_cost::TEXT, notes, actor_id, created_at
	`, lotID, locationID, qty, movType, refID, idempotencyKey, unitCost, notes, actorID,
	).Scan(&mov.ID, &mov.LotID, &mov.LocationID, &qtyStr, &mov.MovementType,
		&mov.ReferenceID, &unitCostStr, &mov.Notes, &mov.ActorID, &mov.CreatedAt)
	if err != nil {
		return nil, err
	}
	if qtyStr != nil {
		mov.Quantity = parseDecimal(*qtyStr)
	}
	mov.UnitCost = parseNullDecimal(unitCostStr)
	return mov, nil
}

func scanItems(rows pgx.Rows) ([]Item, error) {
	var items []Item
	for rows.Next() {
		var item Item
		var rawMinQty *string
		var attrs []byte
		if err := rows.Scan(
			&item.ID, &item.SKU, &item.Name, &item.Description, &item.Unit, &item.Category,
			&rawMinQty, &attrs, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.MinStockQty = parseNullDecimal(rawMinQty)
		item.Attributes = attrs
		items = append(items, item)
	}
	return items, rows.Err()
}

func parseDecimal(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func parseNullDecimal(s *string) *decimal.Decimal {
	if s == nil {
		return nil
	}
	d, _ := decimal.NewFromString(*s)
	return &d
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
