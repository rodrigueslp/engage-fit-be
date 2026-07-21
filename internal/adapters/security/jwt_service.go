package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

type JWTService struct {
	secret string
}

func NewJWTService(secret string) JWTService {
	return JWTService{secret: secret}
}

func (s JWTService) Generate(ctx context.Context, claims services.AuthClaims) (string, error) {
	now := time.Now()
	header := jwtHeader{Algorithm: "HS256", Type: "JWT"}
	payload := jwtPayload{
		UserID:      string(claims.UserID),
		BoxID:       string(claims.BoxID),
		Role:        string(claims.Role),
		AuthVersion: claims.AuthVersion,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(24 * time.Hour).Unix(),
	}

	headerPart, err := encodeJSON(header)
	if err != nil {
		return "", err
	}
	payloadPart, err := encodeJSON(payload)
	if err != nil {
		return "", err
	}

	unsigned := headerPart + "." + payloadPart
	signature := s.sign(unsigned)
	return unsigned + "." + signature, nil
}

func (s JWTService) Validate(ctx context.Context, token string) (*services.AuthClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token")
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSignature := s.sign(unsigned)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return nil, errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	if payload.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("expired token")
	}

	return &services.AuthClaims{
		UserID:      domain.ID(payload.UserID),
		BoxID:       domain.ID(payload.BoxID),
		Role:        domain.UserRole(payload.Role),
		AuthVersion: payload.AuthVersion,
	}, nil
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtPayload struct {
	UserID      string `json:"sub"`
	BoxID       string `json:"box_id"`
	Role        string `json:"role"`
	AuthVersion int    `json:"auth_version"`
	IssuedAt    int64  `json:"iat"`
	ExpiresAt   int64  `json:"exp"`
}

func encodeJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (s JWTService) sign(unsigned string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
