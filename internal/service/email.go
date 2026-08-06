package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type EmailService struct {
	DB       *sqlx.DB
	AppName  string
	AppURL   string
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	FromEmail string
}

func NewEmailService(db *sqlx.DB, appName, appURL string) *EmailService {
	return &EmailService{
		DB:        db,
		AppName:   appName,
		AppURL:    appURL,
		FromEmail: "noreply@investo.id",
	}
}

func (s *EmailService) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *EmailService) createToken(userID int64, tokenType string) (*model.EmailVerification, error) {
	token := s.generateToken()
	ev := &model.EmailVerification{
		UserID:    userID,
		Token:     token,
		Type:      tokenType,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	query := `INSERT INTO email_verifications (user_id, token, type, expires_at) VALUES (?, ?, ?, ?)`
	if _, err := s.DB.Exec(query, ev.UserID, ev.Token, ev.Type, ev.ExpiresAt); err != nil {
		return nil, fmt.Errorf("EmailService.createToken: %w", err)
	}
	return ev, nil
}

func (s *EmailService) VerifyToken(token string, tokenType string) (int64, error) {
	var ev model.EmailVerification
	query := `SELECT * FROM email_verifications WHERE token = ? AND type = ? AND used = FALSE AND expires_at > NOW()`
	if err := s.DB.Get(&ev, query, token, tokenType); err != nil {
		return 0, fmt.Errorf("token tidak valid atau sudah kadaluarsa")
	}
	if _, err := s.DB.Exec("UPDATE email_verifications SET used = TRUE WHERE id = ?", ev.ID); err != nil {
		return 0, fmt.Errorf("EmailService.VerifyToken: %w", err)
	}
	return ev.UserID, nil
}

func (s *EmailService) SendVerificationEmail(email string, userID int64) error {
	ev, err := s.createToken(userID, "verify")
	if err != nil {
		return err
	}
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", s.AppURL, ev.Token)
	html := s.buildHTML("Verifikasi Email Anda", fmt.Sprintf(`
		<h2>Verifikasi Email</h2>
		<p>Terima kasih telah mendaftar di %s. Silakan klik tombol di bawah untuk verifikasi email Anda:</p>
		<a href="%s" style="display:inline-block;background:#3b82f6;color:#fff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;margin:16px 0">Verifikasi Email</a>
		<p style="color:#64748b;font-size:13px">Link ini berlaku selama 24 jam. Jika Anda tidak mendaftar di %s, abaikan email ini.</p>
	`, s.AppName, verifyLink, s.AppName))
	s.logEmail(email, "Verifikasi Email", html)
	return nil
}

func (s *EmailService) SendPasswordReset(email string, userID int64) error {
	ev, err := s.createToken(userID, "reset")
	if err != nil {
		return err
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.AppURL, ev.Token)
	html := s.buildHTML("Reset Password", fmt.Sprintf(`
		<h2>Reset Password</h2>
		<p>Anda meminta reset password untuk akun %s Anda. Klik tombol di bawah untuk membuat password baru:</p>
		<a href="%s" style="display:inline-block;background:#3b82f6;color:#fff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;margin:16px 0">Reset Password</a>
		<p style="color:#64748b;font-size:13px">Link ini berlaku selama 24 jam. Jika Anda tidak meminta reset password, abaikan email ini.</p>
	`, s.AppName, resetLink))
	s.logEmail(email, "Reset Password", html)
	return nil
}

func (s *EmailService) SendWelcomeEmail(email, name string) error {
	html := s.buildHTML(fmt.Sprintf("Selamat Datang di %s, %s!", s.AppName, name), fmt.Sprintf(`
		<h2>Selamat Datang, %s!</h2>
		<p>Akun Anda di <strong>%s</strong> telah berhasil dibuat. Anda sekarang dapat mengakses fitur lengkap platform kami:</p>
		<ul style="text-align:left;padding-left:20px">
			<li>Analisa saham IDX real-time</li>
			<li>Portfolio tracking & risk analysis</li>
			<li>AI-powered signals & predictions</li>
			<li>Forex market intelligence</li>
		</ul>
		<a href="%s/dashboard" style="display:inline-block;background:#3b82f6;color:#fff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;margin:16px 0">Mulai Investasi</a>
		<p style="color:#64748b;font-size:13px">Butuh bantuan? Hubungi tim kami kapan saja.</p>
	`, name, s.AppName, s.AppURL))
	s.logEmail(email, fmt.Sprintf("Selamat Datang di %s", s.AppName), html)
	return nil
}

func (s *EmailService) SendSignalAlert(email, signal string) error {
	html := s.buildHTML("Signal Trading Baru - Investo", fmt.Sprintf(`
		<h2>Signal Trading Baru</h2>
		<div style="background:#1e293b;border:1px solid #334155;border-radius:8px;padding:16px;margin:16px 0;color:#e2e8f0;font-family:monospace;white-space:pre-wrap">%s</div>
		<a href="%s/ai/signal-center" style="display:inline-block;background:#3b82f6;color:#fff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;margin:16px 0">Lihat Detail Signal</a>
		<p style="color:#64748b;font-size:13px">Disclaimer: Ini bukan rekomendasi beli/jual. Lakukan riset mandiri sebelum trading.</p>
	`, signal, s.AppURL))
	s.logEmail(email, "Signal Trading - Investo", html)
	return nil
}

func (s *EmailService) SendForgotPassword(email string, userID int64) error {
	ev, err := s.createToken(userID, "reset")
	if err != nil {
		return err
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.AppURL, ev.Token)
	html := s.buildHTML("Reset Password", fmt.Sprintf(`
		<h2>Reset Password</h2>
		<p>Anda meminta reset password. Klik tombol di bawah:</p>
		<a href="%s" style="display:inline-block;background:#3b82f6;color:#fff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;margin:16px 0">Reset Password</a>
	`, resetLink))
	s.logEmail(email, "Reset Password - Investo", html)
	return nil
}

func (s *EmailService) logEmail(to, subject, html string) {
	log.Printf("[EMAIL] To: %s | Subject: %s", to, subject)
	if len(html) > 500 {
		log.Printf("[EMAIL] Body preview: %s...", strings.ReplaceAll(html[:500], "\n", " "))
	} else {
		log.Printf("[EMAIL] Body: %s", strings.ReplaceAll(html, "\n", " "))
	}
	if s.SMTPHost != "" {
		log.Printf("[EMAIL] Would send via SMTP %s:%s", s.SMTPHost, s.SMTPPort)
	}
}

func (s *EmailService) buildHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#0b1120;font-family:'Inter',system-ui,sans-serif;font-size:14px;color:#e2e8f0">
<div style="max-width:560px;margin:40px auto;background:#1e293b;border:1px solid #334155;border-radius:14px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#1e40af,#3b82f6);padding:32px;text-align:center">
    <div style="font-size:28px;font-weight:800;color:#fff">%s</div>
    <div style="font-size:16px;color:#bfdbfe;margin-top:4px">%s</div>
  </div>
  <div style="padding:32px;text-align:center">
    %s
  </div>
  <div style="background:#0f172a;padding:16px;text-align:center;font-size:11px;color:#475569">
    &copy; 2026 %s &middot; Platform Analisa Saham & Forex Indonesia<br>
    Investasi mengandung risiko. Data disediakan untuk edukasi.
  </div>
</div>
</body>
</html>`, s.AppName, title, body, s.AppName)
}

func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
