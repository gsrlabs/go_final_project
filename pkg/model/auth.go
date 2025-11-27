package model

import(
	"github.com/golang-jwt/jwt/v5"
)

// SignInInput - структура для входа
type SignInInput struct {
    Password string `json:"password"`
}

// SignInResponse - ответ при успешной аутентификации
type SignInResponse struct {
    Token string `json:"token"`
}

// Claims - структура для JWT токена
type Claims struct {
    PasswordHash string `json:"pwd_hash"`
    jwt.RegisteredClaims
}