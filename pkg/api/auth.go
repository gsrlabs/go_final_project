package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"todo/pkg/model"

	"github.com/golang-jwt/jwt/v5"
)

// secretKey - JWT signing key
var secretKey = []byte("secret-key")

// generatePasswordHash creates SHA256 hash of password
func generatePasswordHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// SignInHandler handles user authentication
// POST /api/signin
func SignInHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Received signin request from IP %s", r.RemoteAddr)

	if r.Method != http.MethodPost {
		log.Printf("WARN: Invalid method for signin: %s", r.Method)
		sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input model.SignInInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("WARN: Invalid JSON in signin request: %v", err)
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Get password from environment variable
	envPassword := os.Getenv("TODO_PASSWORD")
	if envPassword == "" {
		log.Printf("ERROR: TODO_PASSWORD environment variable not set")
		sendError(w, "authentication not configured", http.StatusInternalServerError)
		return
	}

	// Validate password
	if input.Password != envPassword {
		log.Printf("WARN: Failed login attempt from IP %s", r.RemoteAddr)
		sendError(w, "invalid password", http.StatusUnauthorized)
		return
	}

	log.Printf("INFO: User authenticated successfully from IP %s", r.RemoteAddr)

	// Create JWT token
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
		log.Printf("ERROR: Failed to generate JWT token: %v", err)
		sendError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	log.Printf("DEBUG: JWT token generated and cookie set, expires: %s", expirationTime.Format(time.RFC3339))

	// Set HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
		Path:     "/",
	})

	 //TODO Print token for testing purposes (remove in production)
	fmt.Println("***\nToken for testing:\n" + tokenString + "\n***")
	
	sendJSON(w, model.SignInResponse{Token: tokenString})
}


// AuthMiddleware validates JWT tokens for protected routes
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG: AuthMiddleware checking request: %s %s", r.Method, r.URL.Path)


		envPassword := os.Getenv("TODO_PASSWORD")
		if envPassword == "" {
			log.Printf("DEBUG: Authentication disabled, TODO_PASSWORD not set")
			next(w, r)
			return
		}

		// Extract token from cookie or Authorization header
		var tokenString string
		cookie, err := r.Cookie("token")
		if err == nil {
			tokenString = cookie.Value
			log.Printf("DEBUG: Token found in cookie")
		}

		// Fallback to Authorization header
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				log.Printf("DEBUG: Token found in Authorization header")
			}
		}

		if tokenString == "" {
			log.Printf("WARN: No authentication token provided for %s %s from IP %s", r.Method, r.URL.Path, r.RemoteAddr)
			sendError(w, "Authentification required", http.StatusUnauthorized)
			return
		}

		// Parse and validate JWT token
		claims := &model.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return secretKey, nil
		})

		if err != nil {
			log.Printf("WARN: JWT token parsing failed: %v", err)
			sendError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Printf("WARN: Invalid JWT token provided")
			sendError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Check if password has changed (invalidate old tokens)
		currentPasswordHash := generatePasswordHash(envPassword)
		if claims.PasswordHash != currentPasswordHash {
			log.Printf("WARN: JWT token rejected - password changed, IP: %s", r.RemoteAddr)
			sendError(w, "token expired", http.StatusUnauthorized)
			return
		}

		log.Printf("DEBUG: User authenticated successfully for %s %s", r.Method, r.URL.Path)

		next(w, r)
	}
}
