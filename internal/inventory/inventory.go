package inventory

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

var (
	ErrItemNotFound      = errors.New("item not found")
	ErrLocationNotFound  = errors.New("location not found")
	ErrLotNotFound       = errors.New("lot not found")
	ErrDuplicateSKU      = errors.New("SKU already exists")
	ErrDuplicateLot      = errors.New("lot number already exists for this item")
	ErrDuplicateMovement = errors.New("movement with this idempotency key already exists")
)

const (
	MovementReceipt    = "receipt"
	MovementAdjustment = "adjustment"
	MovementTransfer   = "transfer"
)

type Item struct {
	ID          string           `json:"id"`
	SKU         string           `json:"sku"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Unit        string           `json:"unit"`
	Category    *string          `json:"category,omitempty"`
	MinStockQty *decimal.Decimal `json:"min_stock_qty,omitempty"`
	Attributes  json.RawMessage  `json:"attributes"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type Location struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id,omitempty"`
	Type     string  `json:"type"`
	IsActive bool    `json:"is_active"`
}

type Lot struct {
	ID          string          `json:"id"`
	ItemID      string          `json:"item_id"`
	LotNumber   string          `json:"lot_number"`
	SupplierRef *string         `json:"supplier_ref,omitempty"`
	ReceivedAt  time.Time       `json:"received_at"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	Status      string          `json:"status"`
	Attributes  json.RawMessage `json:"attributes"`
}

type StockMovement struct {
	ID           int64            `json:"id"`
	LotID        string           `json:"lot_id"`
	LocationID   string           `json:"location_id"`
	Quantity     decimal.Decimal  `json:"quantity"`
	MovementType string           `json:"movement_type"`
	ReferenceID  *string          `json:"reference_id,omitempty"`
	UnitCost     *decimal.Decimal `json:"unit_cost,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
	ActorID      string           `json:"actor_id"`
	CreatedAt    time.Time        `json:"created_at"`
}

type StockLevel struct {
	LotID        string          `json:"lot_id"`
	LocationID   string          `json:"location_id"`
	Quantity     decimal.Decimal `json:"quantity"`
	ItemID       string          `json:"item_id"`
	ItemName     string          `json:"item_name"`
	ItemSKU      string          `json:"item_sku"`
	Unit         string          `json:"unit"`
	LotNumber    string          `json:"lot_number"`
	LocationCode string          `json:"location_code"`
	LocationName string          `json:"location_name"`
}

// Request types

type CreateItemRequest struct {
	SKU         string           `json:"sku"`
	Name        string           `json:"name"`
	Description *string          `json:"description"`
	Unit        string           `json:"unit"`
	Category    *string          `json:"category"`
	MinStockQty *decimal.Decimal `json:"min_stock_qty"`
	Attributes  json.RawMessage  `json:"attributes"`
}

type UpdateItemRequest struct {
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	Unit        *string          `json:"unit"`
	Category    *string          `json:"category"`
	MinStockQty *decimal.Decimal `json:"min_stock_qty"`
	Attributes  json.RawMessage  `json:"attributes"`
}

type CreateLocationRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
	Type     string  `json:"type"`
}

type CreateLotRequest struct {
	ItemID      string          `json:"item_id"`
	LotNumber   string          `json:"lot_number"`
	SupplierRef *string         `json:"supplier_ref"`
	ReceivedAt  *time.Time      `json:"received_at"`
	ExpiresAt   *time.Time      `json:"expires_at"`
	Attributes  json.RawMessage `json:"attributes"`
}

type RecordMovementRequest struct {
	LotID          string           `json:"lot_id"`
	LocationID     string           `json:"location_id"`
	Quantity       decimal.Decimal  `json:"quantity"`
	MovementType   string           `json:"movement_type"`
	ReferenceID    *string          `json:"reference_id"`
	IdempotencyKey string           `json:"idempotency_key"`
	UnitCost       *decimal.Decimal `json:"unit_cost"`
	Notes          *string          `json:"notes"`
}

type TransferRequest struct {
	LotID            string          `json:"lot_id"`
	SourceLocationID string          `json:"source_location_id"`
	DestLocationID   string          `json:"dest_location_id"`
	Quantity         decimal.Decimal `json:"quantity"`
	IdempotencyKey   string          `json:"idempotency_key"`
	Notes            *string         `json:"notes"`
}
