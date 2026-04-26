package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

// Claims holds the standard + custom JWT claims for access tokens.
type Claims struct {
	jwt.RegisteredClaims
	TenantID   string `json:"tid"`
	TenantSlug string `json:"tslug"`
	Role       string `json:"role"`
}

// IssueAccessToken signs and returns a new access token.
func IssueAccessToken(cfg *config.Config, userID, tenantID, tenantSlug, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg.JWTAccessExpiryMin) * time.Minute)),
			ID:        uuid.New().String(),
		},
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		Role:       role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken verifies the token signature and expiry.
// Returns ErrTokenExpired or ErrTokenInvalid on failure.
func ValidateAccessToken(cfg *config.Config, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// IssueRefreshToken generates a cryptographically random opaque token.
// Returns the plain token (for the cookie) and its SHA-256 hex hash (for the DB).
func IssueRefreshToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	plain = hex.EncodeToString(b)
	sum := sha256.Sum256(b)
	hash = hex.EncodeToString(sum[:])
	return plain, hash, nil
}
