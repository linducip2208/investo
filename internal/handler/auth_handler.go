package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"investo/internal/middleware"
	"investo/internal/repository"
	"investo/internal/service"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type AuthHandler struct {
	AuthService  *service.AuthService
	SessionStore sessions.Store
	Templates    *template.Template
	EmailService *service.EmailService
	TOTPService  *service.TOTPService
	SettingRepo  *repository.SettingRepository
	DB           *sqlx.DB
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Masuk - Investo",
	}
	h.Templates.ExecuteTemplate(w, "auth/login.html", data)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := map[string]interface{}{
			"Title": "Masuk - Investo",
			"Error": "Invalid form data",
		}
		h.Templates.ExecuteTemplate(w, "auth/login.html", data)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		data := map[string]interface{}{
			"Title": "Masuk - Investo",
			"Error": "Email dan password wajib diisi",
		}
		h.Templates.ExecuteTemplate(w, "auth/login.html", data)
		return
	}

	user, err := h.AuthService.Login(email, password)
	if err != nil {
		data := map[string]interface{}{
			"Title": "Masuk - Investo",
			"Error": err.Error(),
		}
		h.Templates.ExecuteTemplate(w, "auth/login.html", data)
		return
	}

	session, _ := h.SessionStore.Get(r, "investo-session")
	session.Values["user_id"] = user.ID
	returnTo, _ := session.Values["return_to"].(string)
	delete(session.Values, "return_to")
	session.Save(r, w)

	if returnTo != "" && returnTo != "/login" && returnTo != "/register" {
		http.Redirect(w, r, returnTo, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Daftar - Investo",
		"Name":  "",
		"Email": "",
	}
	h.Templates.ExecuteTemplate(w, "auth/register.html", data)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := map[string]interface{}{
			"Title": "Daftar - Investo",
			"Error": "Invalid form data",
		}
		h.Templates.ExecuteTemplate(w, "auth/register.html", data)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	if name == "" || email == "" || password == "" {
		data := map[string]interface{}{
			"Title": "Daftar - Investo",
			"Error": "Semua field wajib diisi",
		}
		h.Templates.ExecuteTemplate(w, "auth/register.html", data)
		return
	}

	if password != confirm {
		data := map[string]interface{}{
			"Title": "Daftar - Investo",
			"Error": "Password dan konfirmasi tidak cocok",
		}
		h.Templates.ExecuteTemplate(w, "auth/register.html", data)
		return
	}

	user, err := h.AuthService.Register(name, email, password)
	if err != nil {
		data := map[string]interface{}{
			"Title": "Daftar - Investo",
			"Error": err.Error(),
		}
		h.Templates.ExecuteTemplate(w, "auth/register.html", data)
		return
	}

	session, _ := h.SessionStore.Get(r, "investo-session")
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := h.SessionStore.Get(r, "investo-session")
	session.Values["user_id"] = nil
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title": "Profil - Investo",
		"User":  user,
	}
	h.Templates.ExecuteTemplate(w, "user/profile.html", data)
}

func (h *AuthHandler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/dashboard/profile", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")

	err := h.AuthService.UpdateProfile(user.ID, name, email, currentPassword, newPassword)
	if err != nil {
		data := map[string]interface{}{
			"Title": "Profil - Investo",
			"User":  user,
			"Error": err.Error(),
		}
		h.Templates.ExecuteTemplate(w, "user/profile.html", data)
		return
	}

	session, _ := h.SessionStore.Get(r, "investo-session")
	session.AddFlash("Profil berhasil diperbarui")
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard/profile", http.StatusSeeOther)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		data := map[string]interface{}{
			"Title": "Verifikasi Email - Investo",
			"Error": "Token tidak ditemukan",
		}
		h.Templates.ExecuteTemplate(w, "auth/verify-email.html", data)
		return
	}

	userID, err := h.EmailService.VerifyToken(token, "verify")
	if err != nil {
		data := map[string]interface{}{
			"Title":   "Verifikasi Email - Investo",
			"Error":   err.Error(),
			"Success": false,
		}
		h.Templates.ExecuteTemplate(w, "auth/verify-email.html", data)
		return
	}

	if _, err := h.DB.Exec("UPDATE users SET email_verified_at = NOW() WHERE id = ?", userID); err != nil {
		log.Printf("VerifyEmail: update users error: %v", err)
	}

	data := map[string]interface{}{
		"Title":   "Verifikasi Email - Investo",
		"Success": true,
	}
	h.Templates.ExecuteTemplate(w, "auth/verify-email.html", data)
}

func (h *AuthHandler) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Lupa Password - Investo",
	}
	h.Templates.ExecuteTemplate(w, "auth/forgot-password.html", data)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := map[string]interface{}{"Title": "Lupa Password - Investo", "Error": "Invalid form data"}
		h.Templates.ExecuteTemplate(w, "auth/forgot-password.html", data)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		data := map[string]interface{}{"Title": "Lupa Password - Investo", "Error": "Email wajib diisi"}
		h.Templates.ExecuteTemplate(w, "auth/forgot-password.html", data)
		return
	}

	user, err := h.AuthService.UserRepo.FindByEmail(email)
	if err != nil {
		data := map[string]interface{}{
			"Title":   "Lupa Password - Investo",
			"Success": true,
			"Message": "Jika email terdaftar, link reset password akan dikirim.",
		}
		h.Templates.ExecuteTemplate(w, "auth/forgot-password.html", data)
		return
	}

	if err := h.EmailService.SendForgotPassword(email, user.ID); err != nil {
		log.Printf("ForgotPassword: send email error: %v", err)
	}

	data := map[string]interface{}{
		"Title":   "Lupa Password - Investo",
		"Success": true,
		"Message": "Jika email terdaftar, link reset password akan dikirim.",
	}
	h.Templates.ExecuteTemplate(w, "auth/forgot-password.html", data)
}

func (h *AuthHandler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	data := map[string]interface{}{
		"Title": "Reset Password - Investo",
		"Token": token,
	}
	h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := map[string]interface{}{"Title": "Reset Password - Investo", "Error": "Invalid form data"}
		h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	if token == "" || password == "" {
		data := map[string]interface{}{"Title": "Reset Password - Investo", "Error": "Semua field wajib diisi"}
		h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
		return
	}

	if password != confirm {
		data := map[string]interface{}{"Title": "Reset Password - Investo", "Error": "Password tidak cocok"}
		h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
		return
	}

	userID, err := h.EmailService.VerifyToken(token, "reset")
	if err != nil {
		data := map[string]interface{}{"Title": "Reset Password - Investo", "Error": err.Error()}
		h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
		return
	}

	if err := h.AuthService.UpdateProfile(userID, "", "", "", password); err != nil {
		data := map[string]interface{}{"Title": "Reset Password - Investo", "Error": "Gagal mengupdate password"}
		h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
		return
	}

	data := map[string]interface{}{
		"Title":   "Reset Password - Investo",
		"Success": true,
		"Message": "Password berhasil direset. Silakan login.",
	}
	h.Templates.ExecuteTemplate(w, "auth/reset-password.html", data)
}

func (h *AuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/login?error=google_auth_failed", http.StatusSeeOther)
		return
	}

	clientID, _ := h.SettingRepo.Get("oauth_google_client_id")
	clientSecret, _ := h.SettingRepo.Get("oauth_google_client_secret")

	if clientID == "" || clientSecret == "" {
		http.Redirect(w, r, "/login?error=google_not_configured", http.StatusSeeOther)
		return
	}

	tokenURL := "https://oauth2.googleapis.com/token"
	form := url.Values{
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {fmt.Sprintf("%s/api/auth/google/callback", h.getAppURL(r))},
		"grant_type":    {"authorization_code"},
	}

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		log.Printf("[Google OAuth] token exchange error: %v", err)
		http.Redirect(w, r, "/login?error=google_token", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &tokenRes)

	if tokenRes.Error != "" {
		log.Printf("[Google OAuth] token error: %s", tokenRes.Error)
		http.Redirect(w, r, "/login?error=google_token", http.StatusSeeOther)
		return
	}

	userReq, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		log.Printf("[Google OAuth] userinfo error: %v", err)
		http.Redirect(w, r, "/login?error=google_userinfo", http.StatusSeeOther)
		return
	}
	defer userResp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	body, _ = io.ReadAll(userResp.Body)
	json.Unmarshal(body, &googleUser)

	if googleUser.Email == "" {
		http.Redirect(w, r, "/login?error=google_no_email", http.StatusSeeOther)
		return
	}

	user, _ := h.AuthService.UserRepo.FindByEmail(googleUser.Email)
	if user == nil {
		user, err = h.AuthService.Register(googleUser.Name, googleUser.Email, "")
		if err != nil {
			log.Printf("[Google OAuth] register error: %v", err)
			http.Redirect(w, r, "/login?error=google_register", http.StatusSeeOther)
			return
		}
	}

	if _, err := h.DB.Exec("UPDATE users SET google_id = ? WHERE id = ?", googleUser.ID, user.ID); err != nil {
		log.Printf("[Google OAuth] update google_id error: %v", err)
	}

	session, _ := h.SessionStore.Get(r, "investo-session")
	session.Values["user_id"] = user.ID
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	secret, qrURL, err := h.TOTPService.GenerateSecret(user.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "gagal generate 2FA secret")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"secret":  secret,
		"qr_url":  qrURL,
	})
}

func (h *AuthHandler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	valid, err := h.TOTPService.VerifyTOTP(user.ID, payload.Code)
	if err != nil || !valid {
		writeJSONError(w, http.StatusBadRequest, "kode TOTP tidak valid")
		return
	}

	if err := h.TOTPService.Enable2FA(user.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "gagal mengaktifkan 2FA")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (h *AuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.TOTPService.Disable2FA(user.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "gagal menonaktifkan 2FA")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (h *AuthHandler) getAppURL(r *http.Request) string {
	appURL, _ := h.SettingRepo.Get("app_url")
	if appURL != "" {
		return appURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
