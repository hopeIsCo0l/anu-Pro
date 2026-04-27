package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Signup(r.Context(), req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if !errors.Is(err, ErrEmailTaken) && !errors.Is(err, ErrSlugTaken) {
			slog.ErrorContext(r.Context(), "signup error", "err", err)
		}
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Login(r.Context(), req, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if !errors.Is(err, ErrBadCredentials) && !errors.Is(err, ErrUserInactive) {
			slog.ErrorContext(r.Context(), "login error", "err", err)
		}
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	resp, err := h.svc.Refresh(r.Context(), body.RefreshToken, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if !errors.Is(err, ErrTokenRevoked) {
			slog.ErrorContext(r.Context(), "refresh error", "err", err)
		}
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	if err := h.svc.Logout(r.Context(), body.RefreshToken); err != nil {
		slog.ErrorContext(r.Context(), "logout error", "err", err)
		writeError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already registered")
	case errors.Is(err, ErrSlugTaken):
		writeError(w, http.StatusConflict, "tenant slug already taken")
	case errors.Is(err, ErrBadCredentials):
		writeError(w, http.StatusUnauthorized, "invalid email or password")
	case errors.Is(err, ErrTokenRevoked):
		writeError(w, http.StatusUnauthorized, "refresh token expired or revoked")
	case errors.Is(err, ErrUserInactive):
		writeError(w, http.StatusForbidden, "account inactive")
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
