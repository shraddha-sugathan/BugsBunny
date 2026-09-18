package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// 1. Define the Context Key
type contextKey string
const claimsKey contextKey = "claims"

// 2. Define the JWT Secret (matches what handlers_users.go is looking for)
var jwtSecret = []byte("your_super_secret_key") // Use your actual secret if different

// 3. Define the JWT Claims struct
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// 4. Authentication Middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("[AUTH FAIL] Missing Authorization header")
			http.Error(w, "Unauthorized: missing header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("[AUTH FAIL] Invalid header format: '%s'", authHeader)
			http.Error(w, "Unauthorized: bad format", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			log.Printf("[AUTH FAIL] Token parse/validation error: %v", err)
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Println("[AUTH FAIL] Token is not valid")
			http.Error(w, "Unauthorized: token not valid", http.StatusUnauthorized)
			return
		}

		log.Printf("[AUTH OK] Request authenticated for User ID: %d", claims.UserID)
		// Proceed to handler...
		log.Printf("[AUTH OK] Request authenticated for User ID: %d", claims.UserID)
                
                // Inject claims into request context
                ctx := context.WithValue(r.Context(), claimsKey, claims)
                next.ServeHTTP(w, r.WithContext(ctx))
		//next.ServeHTTP(w, r)
	}
}

// 5. Admin Authorization Middleware
func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(claimsKey).(*Claims)
		if !ok || (strings.ToLower(claims.Role) != "admin" && strings.ToLower(claims.Role) != "administrator") {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}