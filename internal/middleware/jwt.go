package middleware

import (
	"context"
	"net/http"
	"strings"

	"investo/internal/service"
)

type jwtContextKey string

const JWTClaimsKey jwtContextKey = "jwt_claims"

func JWTAuth(jwtService *service.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"data":{"error":"missing authorization header"},"timestamp":""}`))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"data":{"error":"invalid authorization format"},"timestamp":""}`))
				return
			}

			tokenStr := parts[1]
			claims, err := jwtService.ValidateToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"data":{"error":"invalid or expired token"},"timestamp":""}`))
				return
			}

			ctx := context.WithValue(r.Context(), JWTClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetJWTClaims(r *http.Request) *service.Claims {
	claims, ok := r.Context().Value(JWTClaimsKey).(*service.Claims)
	if !ok {
		return nil
	}
	return claims
}
