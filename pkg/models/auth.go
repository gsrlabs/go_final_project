package models

import (
	"github.com/golang-jwt/jwt/v5"
)

// SignInInput - the structure for entry
type SignInInput struct {
	Password string `json:"password"`
}

// SignInResponse - response upon successful authentication
type SignInResponse struct {
	Token string `json:"token"`
}

// Claims - the structure for the JWT token
type Claims struct {
	PasswordHash string `json:"pwd_hash"`
	jwt.RegisteredClaims
}
