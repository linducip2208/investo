package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Iat    int64  `json:"iat"`
	Exp    int64  `json:"exp"`
}

type JWTService struct {
	Secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{Secret: []byte(secret)}
}

func (s *JWTService) GenerateToken(userID int64, email string) (string, error) {
	now := time.Now().Unix()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Iat:    now,
		Exp:    now + 86400,
	}

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("jwt marshal header: %w", err)
	}
	headerEnc := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("jwt marshal payload: %w", err)
	}
	payloadEnc := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := headerEnc + "." + payloadEnc
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sig, nil
}

func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerEnc, payloadEnc, sigEnc := parts[0], parts[1], parts[2]

	sigBytes, err := base64.RawURLEncoding.DecodeString(sigEnc)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding")
	}

	signingInput := headerEnc + "." + payloadEnc
	mac := hmac.New(sha256.New, s.Secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expectedSig) {
		return nil, fmt.Errorf("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadEnc)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding")
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload json")
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}
