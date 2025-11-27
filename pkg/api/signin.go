package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"todo/pkg/model"

	"github.com/golang-jwt/jwt/v5"
)

// secretKey - ключ для подписи JWT (можно использовать пароль или фиксированный ключ)
var secretKey = []byte("secret-key")

// generatePasswordHash - создает хэш пароля
func generatePasswordHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// SignInHandler - обработчик входа
func SignInHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input model.SignInInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Получаем пароль из переменных окружения
	envPassword := os.Getenv("TODO_PASSWORD")
	if envPassword == "" {
		sendError(w, "authentication not configured", http.StatusInternalServerError)
		return
	}

	// Проверяем пароль
	if input.Password != envPassword {
		sendError(w, "invalid password", http.StatusUnauthorized)
		return
	}

	// Создаем JWT токен
	expirationTime := time.Now().Add(8 * time.Hour)
	claims := &model.Claims{
		PasswordHash: generatePasswordHash(envPassword),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		sendError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Устанавливаем куку
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
		Path:     "/",
	})

	fmt.Println(tokenString)
	
	// Возвращаем токен в JSON
	sendJSON(w, model.SignInResponse{Token: tokenString})
}
