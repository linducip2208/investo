package middleware

import (
	"context"
	"net/http"
	"strings"

	"investo/internal/model"
	"investo/internal/repository"

	"github.com/gorilla/sessions"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	SessionStore sessions.Store
	UserRepo     *repository.UserRepository
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session, err := m.sessionUser(r)
		if err != nil || user == nil {
			if session != nil {
				session.Values["return_to"] = r.URL.RequestURI()
				_ = session.Save(r, w)
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAPIAccess keeps read-only market data public, but requires a valid
// session for state changes and user-scoped data. Admin API paths additionally
// require the admin role.
func (m *AuthMiddleware) RequireAPIAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicAPIRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		user, _, err := m.sessionUser(r)
		if err != nil || user == nil {
			writeAPIError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/admin/") && user.Role != "admin" {
			writeAPIError(w, http.StatusForbidden, "admin role required")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) sessionUser(r *http.Request) (*model.User, *sessions.Session, error) {
	session, err := m.SessionStore.Get(r, "investo-session")
	if err != nil {
		return nil, session, err
	}
	userID, ok := session.Values["user_id"]
	if !ok || userID == nil {
		return nil, session, nil
	}
	id, ok := userID.(int64)
	if !ok {
		return nil, session, nil
	}
	user, err := m.UserRepo.FindByID(id)
	if err != nil {
		return nil, session, err
	}
	return user, session, nil
}

func isPublicAPIRequest(r *http.Request) bool {
	path := r.URL.Path
	if path == "/api/payment/callback" || path == "/api/webhooks/alert" {
		return true
	}
	if r.Method == http.MethodPost {
		switch path {
		case "/api/thesis/backtest", "/api/forex/swap", "/api/forex/margin", "/api/strategies/backtest", "/api/kelly/calculate":
			return true
		default:
			return false
		}
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
		return false
	}

	privatePrefixes := []string{
		"/api/admin/", "/api/alerts", "/api/invoice/", "/api/subscription/",
		"/api/trading/", "/api/risk/", "/api/portfolio/", "/api/portfolios/",
		"/api/export/portfolios/", "/api/ai/usage", "/api/ai/report",
		"/api/ai/providers", "/api/ai/mandates", "/api/ai/approvals",
		"/api/ai/tax-optimizer/", "/api/ai/haiku/", "/api/ai/compliance/",
		"/api/ai/client-report/", "/api/ai/calendar-sync",
	}
	for _, prefix := range privatePrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"success":false,"error":"` + message + `"}`))
}

func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil || user.Role != "admin" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403 Forbidden"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) RequireGuest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.SessionStore.Get(r, "investo-session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		userID, ok := session.Values["user_id"]
		if ok && userID != nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetUser(r *http.Request) *model.User {
	user, ok := r.Context().Value(UserContextKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}

func IsAuthenticated(r *http.Request) bool {
	return GetUser(r) != nil
}
