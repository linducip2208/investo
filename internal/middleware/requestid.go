package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

// RequestID ensures every request carries an X-Request-ID: it propagates an
// inbound value when present, otherwise generates a random 128-bit hex ID.
// The value is set on the response header and stored in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				r2 := r.WithContext(context.WithValue(r.Context(), requestIDKey{}, ""))
				next.ServeHTTP(w, r2)
				return
			}
			rid = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", rid)
		ctx := context.WithValue(r.Context(), requestIDKey{}, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID returns the request ID stored by the RequestID middleware.
func GetRequestID(r *http.Request) string {
	if v, ok := r.Context().Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}
