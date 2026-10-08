package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In production the allowed origin is pinned to INVESTO_APP_URL: echo
		// the request Origin only when its host (and scheme, when configured)
		// matches. Without INVESTO_APP_URL keep "*" for local development.
		// Credentials are never allowed.
		if appURL := strings.TrimSpace(os.Getenv("INVESTO_APP_URL")); appURL == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin := r.Header.Get("Origin"); origin != "" && isAllowedOrigin(origin, appURL) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin reports whether the request Origin matches the configured
// app URL host. The scheme must also match when the app URL carries one.
func isAllowedOrigin(origin, appURL string) bool {
	o, err := url.Parse(origin)
	if err != nil || o.Host == "" {
		return false
	}
	a, err := url.Parse(appURL)
	if err != nil {
		return false
	}
	appHost := a.Host
	if appHost == "" {
		appHost = a.Path
	}
	if appHost == "" || !strings.EqualFold(o.Host, appHost) {
		return false
	}
	if a.Scheme != "" && o.Scheme != "" && !strings.EqualFold(o.Scheme, a.Scheme) {
		return false
	}
	return true
}

// CSRFOrigin rejects cross-origin browser mutations that rely on the session
// cookie. Bearer-token API calls and explicitly listed inbound callbacks do not
// use cookie authentication and are therefore outside CSRF's threat model.
func CSRFOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) || csrfExemptPath(r.URL.Path) || bearerAuthenticated(r) {
			next.ServeHTTP(w, r)
			return
		}

		_, sessionErr := r.Cookie("investo-session")
		hasSession := sessionErr == nil
		contentType := strings.ToLower(r.Header.Get("Content-Type"))
		isBrowserForm := strings.HasPrefix(contentType, "application/x-www-form-urlencoded") ||
			strings.HasPrefix(contentType, "multipart/form-data")
		if !hasSession && !isBrowserForm {
			next.ServeHTTP(w, r)
			return
		}

		if !sameOriginRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "cross-origin request rejected"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions || method == http.MethodTrace
}

func csrfExemptPath(path string) bool {
	return path == "/api/payment/callback" || path == "/api/webhooks/alert"
}

func bearerAuthenticated(r *http.Request) bool {
	parts := strings.Fields(r.Header.Get("Authorization"))
	return len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != ""
}

func sameOriginRequest(r *http.Request) bool {
	source := r.Header.Get("Origin")
	if source == "" {
		source = r.Header.Get("Referer")
	}
	if source == "" {
		return false
	}
	u, err := url.Parse(source)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func StackRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
