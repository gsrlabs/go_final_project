package api

import (
	"net/http"
	"os"
	"strings"
	"todo/pkg/model"

	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware - middleware для проверки аутентификации
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, требуется ли аутентификация
		envPassword := os.Getenv("TODO_PASSWORD")
		if envPassword == "" {
			// Аутентификация не требуется
			next(w, r)
			return
		}

		// Получаем токен из куки
		var tokenString string
		cookie, err := r.Cookie("token")
		if err == nil {
			tokenString = cookie.Value
		}

		// Если нет токена в куках, проверяем заголовок Authorization
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			sendError(w, "Authentification required", http.StatusUnauthorized)
			return
		}

		// Парсим и проверяем токен
		claims := &model.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return secretKey, nil
		})

		if err != nil || !token.Valid {
			sendError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Проверяем, что хэш пароля в токене соответствует текущему паролю
		currentPasswordHash := generatePasswordHash(envPassword)
		if claims.PasswordHash != currentPasswordHash {
			sendError(w, "token expired", http.StatusUnauthorized)
			return
		}

		if claims.PasswordHash != currentPasswordHash {
			sendError(w, "password changed, please login again", http.StatusUnauthorized)
			return
		}

		// Токен валиден, передаем управление следующему обработчику
		next(w, r)
	}
}
