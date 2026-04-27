package recipe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"time"
)

// ── Errors ────────────────────────────────────────────────────────────────────

var (
	ErrRecipeNotFound  = errors.New("recipe not found")
	ErrVersionNotFound = errors.New("recipe version not found")
	ErrInputNotFound   = errors.New("recipe input not found")
	ErrOutputNotFound  = errors.New("recipe output not found")
)

// ── Domain types ──────────────────────────────────────────────────────────────

type Recipe struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OutputItemID string    `json:"output_item_id"`
	OutputItem   *string   `json:"output_item_name,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LatestVersion *Version `json:"latest_version,omitempty"`
}

type Version struct {
	ID         string          `json:"id"`
	RecipeID   string          `json:"recipe_id"`
	Version    int             `json:"version"`
	OutputQty  decimal.Decimal `json:"output_qty"`
	OutputUnit string          `json:"output_unit"`
	YieldPct   decimal.Decimal `json:"yield_pct"`
	Notes      *string         `json:"notes,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	Inputs     []Input         `json:"inputs,omitempty"`
	Outputs    []Output        `json:"outputs,omitempty"`
}

type Input struct {
	ID               string          `json:"id"`
	RecipeVersionID  string          `json:"recipe_version_id"`
	ItemID           string          `json:"item_id"`
	ItemName         *string         `json:"item_name,omitempty"`
	ItemSKU          *string         `json:"item_sku,omitempty"`
	Quantity         decimal.Decimal `json:"quantity"`
	Unit             string          `json:"unit"`
	IsOptional       bool            `json:"is_optional"`
	Notes            *string         `json:"notes,omitempty"`
	SortOrder        int             `json:"sort_order"`
}

type Output struct {
	ID               string          `json:"id"`
	RecipeVersionID  string          `json:"recipe_version_id"`
	ItemID           string          `json:"item_id"`
	ItemName         *string         `json:"item_name,omitempty"`
	Quantity         decimal.Decimal `json:"quantity"`
	Unit             string          `json:"unit"`
	OutputType       string          `json:"output_type"`
	Notes            *string         `json:"notes,omitempty"`
}

// ── Request types ─────────────────────────────────────────────────────────────

type CreateRecipeRequest struct {
	Name         string              `json:"name"`
	OutputItemID string              `json:"output_item_id"`
	Notes        *string             `json:"notes"`
	Version      CreateVersionConfig `json:"version"`
}

type CreateVersionConfig struct {
	OutputQty  decimal.Decimal       `json:"output_qty"`
	OutputUnit string                `json:"output_unit"`
	YieldPct   *decimal.Decimal      `json:"yield_pct"`
	Notes      *string               `json:"notes"`
	Inputs     []CreateInputRequest  `json:"inputs"`
	Outputs    []CreateOutputRequest `json:"outputs"`
}

type CreateVersionRequest struct {
	OutputQty  decimal.Decimal       `json:"output_qty"`
	OutputUnit string                `json:"output_unit"`
	YieldPct   *decimal.Decimal      `json:"yield_pct"`
	Notes      *string               `json:"notes"`
	Inputs     []CreateInputRequest  `json:"inputs"`
	Outputs    []CreateOutputRequest `json:"outputs"`
}

type CreateInputRequest struct {
	ItemID     string          `json:"item_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Unit       string          `json:"unit"`
	IsOptional bool            `json:"is_optional"`
	Notes      *string         `json:"notes"`
	SortOrder  int             `json:"sort_order"`
}

type CreateOutputRequest struct {
	ItemID     string          `json:"item_id"`
	Quantity   decimal.Decimal `json:"quantity"`
	Unit       string          `json:"unit"`
	OutputType string          `json:"output_type"`
	Notes      *string         `json:"notes"`
}

// ── Service ───────────────────────────────────────────────────────────────────

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) CreateRecipe(ctx context.Context, req CreateRecipeRequest) (*Recipe, error) {
	if req.Name == "" || req.OutputItemID == "" {
		return nil, fmt.Errorf("name and output_item_id are required")
	}
	if req.Version.OutputQty.IsZero() || req.Version.OutputUnit == "" {
		return nil, fmt.Errorf("version.output_qty and version.output_unit are required")
	}

	yieldPct := decimal.NewFromInt(100)
	if req.Version.YieldPct != nil {
		yieldPct = *req.Version.YieldPct
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var recipeID string
	if err := tx.QueryRow(ctx,
		"INSERT INTO recipes (name, output_item_id, notes) VALUES ($1, $2, $3) RETURNING id",
		req.Name, req.OutputItemID, req.Notes,
	).Scan(&recipeID); err != nil {
		return nil, fmt.Errorf("create recipe: %w", err)
	}

	v, err := insertVersion(ctx, tx, recipeID, 1,
		req.Version.OutputQty.String(), req.Version.OutputUnit,
		yieldPct.String(), req.Version.Notes,
		req.Version.Inputs, req.Version.Outputs)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &Recipe{
		ID:            recipeID,
		Name:          req.Name,
		OutputItemID:  req.OutputItemID,
		Notes:         req.Notes,
		IsActive:      true,
		LatestVersion: v,
	}, nil
}

func (s *Service) ListRecipes(ctx context.Context, activeOnly bool) ([]Recipe, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.name, r.output_item_id, i.name, r.notes, r.is_active, r.created_at, r.updated_at,
		       rv.id, rv.version, rv.output_qty::TEXT, rv.output_unit, rv.yield_pct::TEXT
		FROM recipes r
		JOIN items i ON i.id = r.output_item_id
		LEFT JOIN LATERAL (
			SELECT id, version, output_qty, output_unit, yield_pct
			FROM recipe_versions WHERE recipe_id = r.id ORDER BY version DESC LIMIT 1
		) rv ON TRUE
		WHERE ($1::BOOLEAN = FALSE OR r.is_active = TRUE)
		ORDER BY r.name
	`, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list recipes: %w", err)
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var rec Recipe
		var itemName string
		var vID, vUnit *string
		var vNum *int
		var vOutQty, vYield *string
		if err := rows.Scan(
			&rec.ID, &rec.Name, &rec.OutputItemID, &itemName, &rec.Notes, &rec.IsActive,
			&rec.CreatedAt, &rec.UpdatedAt,
			&vID, &vNum, &vOutQty, &vUnit, &vYield,
		); err != nil {
			return nil, err
		}
		rec.OutputItem = &itemName
		if vID != nil {
			rec.LatestVersion = &Version{
				ID:         *vID,
				Version:    *vNum,
				OutputQty:  parseDecimal(*vOutQty),
				OutputUnit: *vUnit,
				YieldPct:   parseDecimal(*vYield),
			}
		}
		recipes = append(recipes, rec)
	}
	return recipes, rows.Err()
}

func (s *Service) GetRecipe(ctx context.Context, id string) (*Recipe, error) {
	rec := &Recipe{}
	var itemName string

	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.name, r.output_item_id, i.name, r.notes, r.is_active, r.created_at, r.updated_at
		FROM recipes r JOIN items i ON i.id = r.output_item_id
		WHERE r.id = $1
	`, id).Scan(&rec.ID, &rec.Name, &rec.OutputItemID, &itemName, &rec.Notes, &rec.IsActive, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRecipeNotFound
		}
		return nil, fmt.Errorf("get recipe: %w", err)
	}
	rec.OutputItem = &itemName

	v, err := s.getLatestVersion(ctx, id)
	if err != nil && !errors.Is(err, ErrVersionNotFound) {
		return nil, err
	}
	rec.LatestVersion = v
	return rec, nil
}

func (s *Service) AddVersion(ctx context.Context, recipeID string, req CreateVersionRequest) (*Version, error) {
	yieldPct := decimal.NewFromInt(100)
	if req.YieldPct != nil {
		yieldPct = *req.YieldPct
	}

	var exists bool
	if err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM recipes WHERE id = $1)", recipeID).Scan(&exists); err != nil || !exists {
		return nil, ErrRecipeNotFound
	}

	var nextVersion int
	if err := s.pool.QueryRow(ctx,
		"SELECT COALESCE(MAX(version), 0) + 1 FROM recipe_versions WHERE recipe_id = $1",
		recipeID,
	).Scan(&nextVersion); err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	v, err := insertVersion(ctx, tx, recipeID, nextVersion,
		req.OutputQty.String(), req.OutputUnit, yieldPct.String(), req.Notes,
		req.Inputs, req.Outputs)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return v, nil
}

func (s *Service) GetVersion(ctx context.Context, recipeID, versionID string) (*Version, error) {
	v := &Version{}
	var outQty, yieldPct string

	err := s.pool.QueryRow(ctx, `
		SELECT id, recipe_id, version, output_qty::TEXT, output_unit, yield_pct::TEXT, notes, created_at
		FROM recipe_versions WHERE id = $1 AND recipe_id = $2
	`, versionID, recipeID).Scan(
		&v.ID, &v.RecipeID, &v.Version, &outQty, &v.OutputUnit, &yieldPct, &v.Notes, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("get version: %w", err)
	}
	v.OutputQty = parseDecimal(outQty)
	v.YieldPct = parseDecimal(yieldPct)

	inputs, err := s.getInputs(ctx, versionID)
	if err != nil {
		return nil, err
	}
	v.Inputs = inputs

	outputs, err := s.getOutputs(ctx, versionID)
	if err != nil {
		return nil, err
	}
	v.Outputs = outputs
	return v, nil
}

func (s *Service) getLatestVersion(ctx context.Context, recipeID string) (*Version, error) {
	var vID string
	err := s.pool.QueryRow(ctx,
		"SELECT id FROM recipe_versions WHERE recipe_id = $1 ORDER BY version DESC LIMIT 1",
		recipeID,
	).Scan(&vID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	return s.GetVersion(ctx, recipeID, vID)
}

func (s *Service) getInputs(ctx context.Context, versionID string) ([]Input, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ri.id, ri.recipe_version_id, ri.item_id, i.name, i.sku,
		       ri.quantity::TEXT, ri.unit, ri.is_optional, ri.notes, ri.sort_order
		FROM recipe_inputs ri
		JOIN items i ON i.id = ri.item_id
		WHERE ri.recipe_version_id = $1
		ORDER BY ri.sort_order, ri.id
	`, versionID)
	if err != nil {
		return nil, fmt.Errorf("get inputs: %w", err)
	}
	defer rows.Close()

	var inputs []Input
	for rows.Next() {
		var inp Input
		var qty string
		if err := rows.Scan(&inp.ID, &inp.RecipeVersionID, &inp.ItemID, &inp.ItemName, &inp.ItemSKU,
			&qty, &inp.Unit, &inp.IsOptional, &inp.Notes, &inp.SortOrder); err != nil {
			return nil, err
		}
		inp.Quantity = parseDecimal(qty)
		inputs = append(inputs, inp)
	}
	return inputs, rows.Err()
}

func (s *Service) getOutputs(ctx context.Context, versionID string) ([]Output, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ro.id, ro.recipe_version_id, ro.item_id, i.name,
		       ro.quantity::TEXT, ro.unit, ro.output_type, ro.notes
		FROM recipe_outputs ro
		JOIN items i ON i.id = ro.item_id
		WHERE ro.recipe_version_id = $1
	`, versionID)
	if err != nil {
		return nil, fmt.Errorf("get outputs: %w", err)
	}
	defer rows.Close()

	var outputs []Output
	for rows.Next() {
		var out Output
		var qty string
		if err := rows.Scan(&out.ID, &out.RecipeVersionID, &out.ItemID, &out.ItemName,
			&qty, &out.Unit, &out.OutputType, &out.Notes); err != nil {
			return nil, err
		}
		out.Quantity = parseDecimal(qty)
		outputs = append(outputs, out)
	}
	return outputs, rows.Err()
}

// ── HTTP Handler ──────────────────────────────────────────────────────────────

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rec, err := h.svc.CreateRecipe(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") != "false"
	recipes, err := h.svc.ListRecipes(r.Context(), activeOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if recipes == nil {
		recipes = []Recipe{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"recipes": recipes})
}

func (h *Handler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	rec, err := h.svc.GetRecipe(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *Handler) AddVersion(w http.ResponseWriter, r *http.Request) {
	var req CreateVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	v, err := h.svc.AddVersion(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		if errors.Is(err, ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.GetVersion(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "vid"))
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func insertVersion(ctx context.Context, tx pgx.Tx, recipeID string, version int,
	outQty, outUnit, yieldPct string, notes *string,
	inputs []CreateInputRequest, outputs []CreateOutputRequest) (*Version, error) {

	v := &Version{RecipeID: recipeID, Version: version}
	var outQtyStr, yieldStr string

	if err := tx.QueryRow(ctx, `
		INSERT INTO recipe_versions (recipe_id, version, output_qty, output_unit, yield_pct, notes)
		VALUES ($1, $2, $3::NUMERIC, $4, $5::NUMERIC, $6)
		RETURNING id, output_qty::TEXT, output_unit, yield_pct::TEXT, notes, created_at
	`, recipeID, version, outQty, outUnit, yieldPct, notes,
	).Scan(&v.ID, &outQtyStr, &v.OutputUnit, &yieldStr, &v.Notes, &v.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert version: %w", err)
	}
	v.OutputQty = parseDecimal(outQtyStr)
	v.YieldPct = parseDecimal(yieldStr)

	for i, inp := range inputs {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO recipe_inputs (recipe_version_id, item_id, quantity, unit, is_optional, notes, sort_order)
			VALUES ($1, $2, $3::NUMERIC, $4, $5, $6, $7)
			RETURNING id
		`, v.ID, inp.ItemID, inp.Quantity.String(), inp.Unit, inp.IsOptional, inp.Notes,
			func() int { if inp.SortOrder != 0 { return inp.SortOrder }; return i }(),
		).Scan(&id); err != nil {
			return nil, fmt.Errorf("insert input: %w", err)
		}
		v.Inputs = append(v.Inputs, Input{
			ID: id, RecipeVersionID: v.ID, ItemID: inp.ItemID,
			Quantity: inp.Quantity, Unit: inp.Unit, IsOptional: inp.IsOptional,
			Notes: inp.Notes,
		})
	}

	for _, out := range outputs {
		if out.OutputType == "" {
			out.OutputType = "co_product"
		}
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO recipe_outputs (recipe_version_id, item_id, quantity, unit, output_type, notes)
			VALUES ($1, $2, $3::NUMERIC, $4, $5, $6)
			RETURNING id
		`, v.ID, out.ItemID, out.Quantity.String(), out.Unit, out.OutputType, out.Notes,
		).Scan(&id); err != nil {
			return nil, fmt.Errorf("insert output: %w", err)
		}
		v.Outputs = append(v.Outputs, Output{
			ID: id, RecipeVersionID: v.ID, ItemID: out.ItemID,
			Quantity: out.Quantity, Unit: out.Unit, OutputType: out.OutputType, Notes: out.Notes,
		})
	}

	return v, nil
}

func parseDecimal(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
