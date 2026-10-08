package middleware

import (
	"net"
	"net/http"
	"strings"

	"investo/internal/repository"
)

func RequireIPWhitelist(settingRepo *repository.SettingRepository) func(http.Handler) http.Handler {
	if settingRepo == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			whitelist, err := settingRepo.Get("admin_ip_whitelist")
			if err != nil || whitelist == "" {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r)
			allowed := false
			for _, ip := range strings.Split(whitelist, ",") {
				ip = strings.TrimSpace(ip)
				if ip == "" {
					continue
				}
				if _, cidr, err := net.ParseCIDR(ip); err == nil {
					if cidr.Contains(net.ParseIP(clientIP)) {
						allowed = true
						break
					}
				} else if ip == clientIP {
					allowed = true
					break
				}
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"IP tidak diizinkan untuk mengakses admin","ip":"` + clientIP + `"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
