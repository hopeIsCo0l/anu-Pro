package inventory

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/tenant"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ── Items ─────────────────────────────────────────────────────────────────────

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.svc.CreateItem(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := optString(q.Get("q"))
	category := optString(q.Get("category"))
	activeOnly := q.Get("active") != "false"
	limit, offset := parsePagination(q)

	items, err := h.svc.ListItems(r.Context(), search, category, activeOnly, limit, offset)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse("items", items))
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetItem(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	item, err := h.svc.UpdateItem(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeactivateItem(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeactivateItem(r.Context(), chi.URLParam(r, "id")); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Locations ─────────────────────────────────────────────────────────────────

func (h *Handler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	loc, err := h.svc.CreateLocation(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, loc)
}

func (h *Handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	locs, err := h.svc.ListLocations(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse("locations", locs))
}

func (h *Handler) GetLocation(w http.ResponseWriter, r *http.Request) {
	loc, err := h.svc.GetLocation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

// ── Lots ──────────────────────────────────────────────────────────────────────

func (h *Handler) CreateLot(w http.ResponseWriter, r *http.Request) {
	var req CreateLotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	lot, err := h.svc.CreateLot(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lot)
}

func (h *Handler) ListLots(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	itemID := optString(q.Get("item_id"))
	limit, offset := parsePagination(q)

	lots, err := h.svc.ListLots(r.Context(), itemID, limit, offset)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse("lots", lots))
}

func (h *Handler) GetLot(w http.ResponseWriter, r *http.Request) {
	lot, err := h.svc.GetLot(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lot)
}

// ── Stock ─────────────────────────────────────────────────────────────────────

func (h *Handler) RecordMovement(w http.ResponseWriter, r *http.Request) {
	var req RecordMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	actorID := tenant.UserIDFromContext(r.Context())

	mov, err := h.svc.RecordMovement(r.Context(), req, actorID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mov)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	actorID := tenant.UserIDFromContext(r.Context())

	movs, err := h.svc.Transfer(r.Context(), req, actorID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"movements": movs})
}

func (h *Handler) GetCurrentStock(w http.ResponseWriter, r *http.Request) {
	itemID := optString(r.URL.Query().Get("item_id"))
	levels, err := h.svc.GetCurrentStock(r.Context(), itemID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse("stock", levels))
}

func (h *Handler) ListMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lotID := optString(q.Get("lot_id"))
	limit, offset := parsePagination(q)

	movs, err := h.svc.ListMovements(r.Context(), lotID, limit, offset)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse("movements", movs))
}

// ── CSV import ────────────────────────────────────────────────────────────────

func (h *Handler) ImportItems(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid multipart form")
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field required")
		return
	}
	defer f.Close()

	result, err := ImportItemsCSV(r.Context(), h.svc, f)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrItemNotFound), errors.Is(err, ErrLocationNotFound), errors.Is(err, ErrLotNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrDuplicateSKU), errors.Is(err, ErrDuplicateLot), errors.Is(err, ErrDuplicateMovement):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func listResponse[T any](key string, items []T) map[string]any {
	if items == nil {
		items = []T{}
	}
	return map[string]any{key: items}
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parsePagination(q interface{ Get(string) string }) (limit, offset int) {
	limit, _ = strconv.Atoi(q.Get("limit"))
	offset, _ = strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	return
}
