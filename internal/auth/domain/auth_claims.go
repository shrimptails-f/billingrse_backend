package domain

import "github.com/golang-jwt/jwt/v5"

// AuthClaims represents JWT payload shared between auth layers.
type AuthClaims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}
