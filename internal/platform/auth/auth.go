package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appcfg "github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/migrator"
)

var (
	ErrEmailTaken     = errors.New("email already registered")
	ErrSlugTaken      = errors.New("tenant slug already taken")
	ErrBadCredentials = errors.New("invalid email or password")
	ErrTokenRevoked   = errors.New("refresh token revoked or expired")
	ErrUserInactive   = errors.New("user account inactive")
)

type Service struct {
	pool *pgxpool.Pool
	cfg  *appcfg.Config
}

func NewService(pool *pgxpool.Pool, cfg *appcfg.Config) *Service {
	return &Service{pool: pool, cfg: cfg}
}

type SignupRequest struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	Password   string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       string `json:"user_id"`
	TenantID     string `json:"tenant_id"`
	TenantSlug   string `json:"tenant_slug"`
	Role         string `json:"role"`
}

func (s *Service) Signup(ctx context.Context, req SignupRequest, userAgent, ipAddr string) (*AuthResponse, error) {
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var slugExists bool
	if err := tx.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)",
		req.TenantSlug,
	).Scan(&slugExists); err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if slugExists {
		return nil, ErrSlugTaken
	}

	var tenantID string
	if err := tx.QueryRow(ctx,
		"INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id",
		req.TenantSlug, req.TenantName,
	).Scan(&tenantID); err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	var userID string
	if err := tx.QueryRow(ctx,
		"INSERT INTO users (tenant_id, email, password_hash, full_name, role) VALUES ($1, $2, $3, $4, 'owner') RETURNING id",
		tenantID, req.Email, passwordHash, req.FullName,
	).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO events (tenant_id, actor_id, event_type, payload) VALUES ($1, $2, 'auth.signup', '{}')",
		tenantID, userID,
	); err != nil {
		return nil, fmt.Errorf("write signup event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit signup: %w", err)
	}

	if err := migrator.ProvisionTenant(s.cfg.DatabaseURL, req.TenantSlug); err != nil {
		return nil, fmt.Errorf("provision tenant schema: %w", err)
	}

	return s.issueTokenPair(ctx, userID, tenantID, req.TenantSlug, "owner", userAgent, ipAddr)
}

func (s *Service) Login(ctx context.Context, req LoginRequest, userAgent, ipAddr string) (*AuthResponse, error) {
	var (
		userID       string
		tenantID     string
		tenantSlug   string
		passwordHash string
		role         string
		isActive     bool
	)

	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.tenant_id, u.password_hash, u.role, u.is_active, t.slug
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.email = $1 AND t.suspended_at IS NULL
	`, req.Email).Scan(&userID, &tenantID, &passwordHash, &role, &isActive, &tenantSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBadCredentials
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !isActive {
		return nil, ErrUserInactive
	}

	ok, err := VerifyPassword(req.Password, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, ErrBadCredentials
	}

	_, _ = s.pool.Exec(ctx, "UPDATE users SET last_login_at = now() WHERE id = $1", userID)

	return s.issueTokenPair(ctx, userID, tenantID, tenantSlug, role, userAgent, ipAddr)
}

func (s *Service) Refresh(ctx context.Context, plainToken, userAgent, ipAddr string) (*AuthResponse, error) {
	tokenHash := hashToken(plainToken)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		userID     string
		tenantID   string
		tenantSlug string
		role       string
		isActive   bool
	)

	err = tx.QueryRow(ctx, `
		SELECT rt.user_id, u.tenant_id, u.role, u.is_active, t.slug
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		JOIN tenants t ON t.id = u.tenant_id
		WHERE rt.token_hash = $1
		  AND rt.revoked_at IS NULL
		  AND rt.expires_at > now()
		  AND t.suspended_at IS NULL
	`, tokenHash).Scan(&userID, &tenantID, &role, &isActive, &tenantSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenRevoked
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	if !isActive {
		return nil, ErrUserInactive
	}

	if _, err := tx.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1",
		tokenHash,
	); err != nil {
		return nil, fmt.Errorf("revoke token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh: %w", err)
	}

	return s.issueTokenPair(ctx, userID, tenantID, tenantSlug, role, userAgent, ipAddr)
}

func (s *Service) Logout(ctx context.Context, plainToken string) error {
	tokenHash := hashToken(plainToken)
	_, err := s.pool.Exec(ctx,
		"UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL",
		tokenHash,
	)
	return err
}

func (s *Service) issueTokenPair(ctx context.Context, userID, tenantID, tenantSlug, role, userAgent, ipAddr string) (*AuthResponse, error) {
	accessToken, err := IssueAccessToken(s.cfg, userID, tenantID, tenantSlug, role)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	plain, hash, err := IssueRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTRefreshExpiryDays) * 24 * time.Hour)

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip_addr)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, hash, expiresAt, userAgent, ipAddr); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: plain,
		ExpiresIn:    s.cfg.JWTAccessExpiryMin * 60,
		UserID:       userID,
		TenantID:     tenantID,
		TenantSlug:   tenantSlug,
		Role:         role,
	}, nil
}

// hashToken recovers the raw bytes from a hex-encoded plain token and returns
// the SHA-256 hash that is stored in the database.
func hashToken(plain string) string {
	raw, err := hex.DecodeString(plain)
	if err != nil {
		// Fallback: hash the string directly (shouldn't happen in practice).
		sum := sha256.Sum256([]byte(plain))
		return hex.EncodeToString(sum[:])
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
