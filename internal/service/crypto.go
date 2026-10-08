package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	encryptionKeyOnce sync.Once
	encryptionKeyVal  []byte
	encryptionWarnOnce sync.Once
)

func warnEncryptionOnce(format string, args ...interface{}) {
	encryptionWarnOnce.Do(func() {
		log.Printf(format, args...)
	})
}

// encryptionKey resolves the 32-byte AES key used by EncryptString/
// DecryptString. INVESTO_ENCRYPTION_KEY accepts either a raw 32-byte value
// or a base64-encoded 32-byte value (standard, URL-safe, padded or not).
// The result is cached for the process lifetime. When empty, a dev-only
// derived key is used and a warning is logged exactly once so existing dev
// databases keep booting.
func encryptionKey() []byte {
	encryptionKeyOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("INVESTO_ENCRYPTION_KEY"))
		if raw == "" {
			raw = strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
		}
		if key, ok := parseEncryptionKey(raw); ok {
			encryptionKeyVal = key
			return
		}
		if raw != "" {
			sum := sha256.Sum256([]byte(raw))
			encryptionKeyVal = sum[:]
			warnEncryptionOnce("[SECURITY] INVESTO_ENCRYPTION_KEY is not a raw 32-byte value nor base64-encoded 32 bytes; derived a key from it. Set a proper 32-byte key.")
			return
		}
		sum := sha256.Sum256([]byte("investo-dev-encryption-key|v1"))
		encryptionKeyVal = sum[:]
		warnEncryptionOnce("[SECURITY] INVESTO_ENCRYPTION_KEY is not set; using a dev-only derived key. Encrypted values (TOTP secrets, etc.) are NOT securely protected. Set INVESTO_ENCRYPTION_KEY (32 random bytes, raw or base64) in production.")
	})
	out := make([]byte, 32)
	copy(out, encryptionKeyVal)
	return out
}

func parseEncryptionKey(raw string) ([]byte, bool) {
	if raw == "" {
		return nil, false
	}
	if len(raw) == 32 {
		return []byte(raw), true
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if decoded, err := enc.DecodeString(raw); err == nil && len(decoded) == 32 {
			return decoded, true
		}
	}
	return nil, false
}

// EncryptString encrypts plaintext with AES-256-GCM using a random nonce.
// The stored value is base64(nonce | ciphertext).
func EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(encryptionKey())
	if err != nil {
		return "", fmt.Errorf("crypto.EncryptString cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.EncryptString gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto.EncryptString rand: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptString reverses EncryptString. For backward compatibility it also
// accepts legacy bare-base64 values (previously stored base64-only
// "encrypted" API keys) and returns their decoded plaintext.
func DecryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if plaintext, err := decryptGCM(ciphertext); err == nil {
		return plaintext, nil
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if decoded, err := enc.DecodeString(ciphertext); err == nil {
			return string(decoded), nil
		}
	}
	return "", fmt.Errorf("crypto.DecryptString: value is neither AES-GCM nor legacy base64")
}

func decryptGCM(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	opened, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(opened), nil
}

// HashToken returns the hex-encoded SHA256 of a token. Only the hash is
// stored server-side; the raw token travels by email and is hashed again
// for comparison.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomToken returns n cryptographically random bytes encoded as
// unpadded base64url. rand failures are propagated, never swallowed.
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto.RandomToken rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DeriveSessionKeys splits one configured secret into two independent
// 32-byte keys: one for cookie authentication (HMAC) and one for cookie
// encryption, so the two roles never share key material.
func DeriveSessionKeys(secret string) (authKey, encKey []byte) {
	authSum := sha256.Sum256([]byte(secret + "|auth"))
	encSum := sha256.Sum256([]byte(secret + "|enc"))
	authKey = make([]byte, len(authSum))
	encKey = make([]byte, len(encSum))
	copy(authKey, authSum[:])
	copy(encKey, encSum[:])
	return authKey, encKey
}
