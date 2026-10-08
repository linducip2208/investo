package middleware

import (
	"encoding/json"
	"net/http"

	"investo/internal/service"
)

func RequirePro(paymentSvc *service.PaymentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
				return
			}

			plan, err := paymentSvc.GetUserPlan(user.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "subscription check failed"})
				return
			}

			if plan == "free" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "upgrade ke Pro untuk akses fitur ini", "plan": plan, "upgrade_url": "/pricing"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireWhitelabel(paymentSvc *service.PaymentService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
				return
			}

			plan, err := paymentSvc.GetUserPlan(user.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "subscription check failed"})
				return
			}

			if plan != "whitelabel" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "upgrade ke Whitelabel untuk akses fitur ini", "plan": plan, "upgrade_url": "/pricing"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r)
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
			return
		}

		if user.Role != "admin" && user.Role != "manager" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "admin or manager role required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetPlanBadge(plan string) map[string]string {
	badges := map[string]map[string]string{
		"free":       {"label": "Free", "color": "bg-slate-600"},
		"pro":        {"label": "Pro", "color": "bg-blue-600"},
		"whitelabel": {"label": "Whitelabel", "color": "bg-purple-600"},
	}
	if badge, ok := badges[plan]; ok {
		return badge
	}
	return map[string]string{"label": "Free", "color": "bg-slate-600"}
}
