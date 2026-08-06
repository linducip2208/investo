package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type TOTPService struct {
	DB      *sqlx.DB
	AppName string
}

func NewTOTPService(db *sqlx.DB, appName string) *TOTPService {
	return &TOTPService{DB: db, AppName: appName}
}

func (s *TOTPService) GenerateSecret(userID int64) (string, string, error) {
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("TOTPService.GenerateSecret rand: %w", err)
	}

	secret := strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes), "=")

	query := `UPDATE users SET twofa_secret = ? WHERE id = ?`
	if _, err := s.DB.Exec(query, secret, userID); err != nil {
		return "", "", fmt.Errorf("TOTPService.GenerateSecret update: %w", err)
	}

	var userEmail string
	if err := s.DB.Get(&userEmail, "SELECT email FROM users WHERE id = ?", userID); err != nil {
		return "", "", fmt.Errorf("TOTPService.GenerateSecret email: %w", err)
	}

	qrURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		s.AppName, userEmail, secret, s.AppName)

	log.Printf("[2FA] Generated secret for user %d", userID)
	return secret, qrURL, nil
}

func (s *TOTPService) VerifyTOTP(userID int64, code string) (bool, error) {
	var secret string
	if err := s.DB.Get(&secret, "SELECT twofa_secret FROM users WHERE id = ?", userID); err != nil {
		return false, fmt.Errorf("TOTPService.VerifyTOTP: %w", err)
	}
	if secret == "" {
		return false, fmt.Errorf("2FA belum di-setup untuk akun ini")
	}

	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false, fmt.Errorf("TOTPService.VerifyTOTP decode: %w", err)
	}

	now := time.Now().Unix()
	for drift := int64(-1); drift <= 1; drift++ {
		if s.totpAt(secretBytes, now+drift*30) == code {
			return true, nil
		}
	}
	return false, nil
}

func (s *TOTPService) Enable2FA(userID int64) error {
	query := `UPDATE users SET twofa_enabled = TRUE WHERE id = ?`
	if _, err := s.DB.Exec(query, userID); err != nil {
		return fmt.Errorf("TOTPService.Enable2FA: %w", err)
	}
	log.Printf("[2FA] Enabled for user %d", userID)
	return nil
}

func (s *TOTPService) Disable2FA(userID int64) error {
	query := `UPDATE users SET twofa_enabled = FALSE, twofa_secret = NULL WHERE id = ?`
	if _, err := s.DB.Exec(query, userID); err != nil {
		return fmt.Errorf("TOTPService.Disable2FA: %w", err)
	}
	log.Printf("[2FA] Disabled for user %d", userID)
	return nil
}

func (s *TOTPService) Is2FAEnabled(userID int64) (bool, error) {
	var enabled bool
	query := `SELECT twofa_enabled FROM users WHERE id = ?`
	if err := s.DB.Get(&enabled, query, userID); err != nil {
		return false, fmt.Errorf("TOTPService.Is2FAEnabled: %w", err)
	}
	return enabled, nil
}

func (s *TOTPService) totpAt(secret []byte, timestamp int64) string {
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(timestamp)/30)

	mac := hmac.New(sha1.New, secret)
	mac.Write(counter)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	binary := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	code := binary % 1000000

	return fmt.Sprintf("%06d", code)
}
