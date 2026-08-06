package middleware

import (
	"context"
	"net/http"

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
		session, err := m.SessionStore.Get(r, "investo-session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, ok := session.Values["user_id"]
		if !ok {
			session.Values["return_to"] = r.URL.String()
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		id, ok := userID.(int64)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := m.UserRepo.FindByID(id)
		if err != nil {
			session.Values["user_id"] = nil
			session.Save(r, w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
