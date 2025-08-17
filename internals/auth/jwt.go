package auth

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleDriver   Role = "driver"
	RoleCustomer Role = "customer"
	RoleKitchen  Role = "kitchen"
)

type Claims struct {
	DeliveryID string `json:"delivery_id"`
	Role       Role   `json:"role"`
	jwt.RegisteredClaims
}

func secret() []byte {

	s := os.Getenv("APP_JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-me"
	}

	return []byte(s)
}

func MakeToken(deliveryID string, role Role, ttl time.Duration) (string, error) {

	claims := Claims{
		DeliveryID: deliveryID,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

func ParseToken(tok string) (*Claims, error) {

	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (interface{}, error) {
		return secret(), nil
	})

	if err != nil || !parsed.Valid {

		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func ParseTokenFromRequest(r *http.Request) (*Claims, error) {

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(auth), "bearer") {

		return nil, errors.New("missing bearer token")

	}

	tok := strings.TrimSpace(auth[len("bearer "):])
	return ParseToken(tok)

}
