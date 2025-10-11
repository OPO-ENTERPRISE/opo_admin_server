package http

import (
	"context"
	"log"
	"net/http"
	"strings"

	"opo_admin_server/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// AuthJWT - Middleware de autenticación JWT
func AuthJWT(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Permitir peticiones OPTIONS sin autenticación (preflight CORS)
			if r.Method == "OPTIONS" {
				log.Printf("✅ AuthJWT Middleware - OPTIONS request, skipping auth")
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			log.Printf("🔍 AuthJWT Middleware - URL: %s", r.URL.Path)
			log.Printf("🔍 AuthJWT Middleware - Authorization header: %s", auth)

			if !strings.HasPrefix(auth, "Bearer ") {
				log.Printf("❌ AuthJWT Middleware - Missing Bearer token")
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}

			tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				log.Printf("❌ AuthJWT Middleware - Invalid token: %v", err)
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			// Extraer información del usuario del token
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				log.Printf("✅ AuthJWT Middleware - Token valid, user: %s", claims["email"])
				// Agregar información del usuario al contexto
				ctx := context.WithValue(r.Context(), "user_id", claims["sub"])
				ctx = context.WithValue(ctx, "user_email", claims["email"])
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}
