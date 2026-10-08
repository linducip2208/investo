package service

import (
	"errors"
	"fmt"
	"investo/internal/model"
	"investo/internal/repository"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrEmailExists        = errors.New("email sudah terdaftar")
	ErrSessionNotFound    = errors.New("session tidak ditemukan")
	ErrWeakPassword       = errors.New("password minimal 8 karakter")
)

type AuthService struct {
	UserRepo          *repository.UserRepository
	SessionRepo       *repository.SessionRepository
	PasswordResetRepo *repository.PasswordResetRepository
}

func (s *AuthService) Register(name, email, password string) (*model.User, error) {
	existing, err := s.UserRepo.FindByEmail(email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
	}

	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("AuthService.Register hash: %w", err)
	}

	user := &model.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     "user",
	}

	id, err := s.UserRepo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("AuthService.Register create: %w", err)
	}

	user.ID = id
	user.Password = ""
	return user, nil
}

// RegisterOAuth creates an account for third-party sign-in (e.g. Google).
// The stored password is a random unguessable value, so password-login is
// impossible without going through the reset-password flow.
func (s *AuthService) RegisterOAuth(name, email string) (*model.User, error) {
	randomPassword, err := RandomToken(48)
	if err != nil {
		return nil, fmt.Errorf("AuthService.RegisterOAuth token: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("AuthService.RegisterOAuth hash: %w", err)
	}

	user := &model.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		Role:     "user",
	}

	id, err := s.UserRepo.Create(user)
	if err != nil {
		return nil, fmt.Errorf("AuthService.RegisterOAuth create: %w", err)
	}

	user.ID = id
	user.Password = ""
	return user, nil
}

func (s *AuthService) Login(email, password string) (*model.User, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	user.Password = ""
	return user, nil
}

func (s *AuthService) GetUserFromSession(r *http.Request) (*model.User, error) {
	userID := r.Context().Value("userID")
	if userID == nil {
		cookie, err := r.Cookie("investo_session")
		if err != nil {
			return nil, ErrSessionNotFound
		}

		user, err := s.UserRepo.FindByEmail(cookie.Value)
		if err != nil {
			return nil, ErrSessionNotFound
		}

		user.Password = ""
		return user, nil
	}

	id, ok := userID.(int64)
	if !ok {
		return nil, ErrSessionNotFound
	}

	user, err := s.UserRepo.FindByID(id)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	user.Password = ""
	return user, nil
}

func (s *AuthService) UpdateProfile(userID int64, name, email, currentPassword, newPassword string) error {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	if currentPassword != "" && newPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
			return errors.New("password saat ini salah")
		}
		if len(newPassword) < 8 {
			return ErrWeakPassword
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return errors.New("gagal mengenkripsi password baru")
		}
		user.Password = string(hashedPassword)
		s.revokeAllSessions(userID)
	}

	user.Name = name
	user.Email = email

	return s.UserRepo.Update(user)
}

// SetPassword replaces a user's password without touching other profile
// fields (used by the reset-password flow). All existing server-side
// sessions for the user are revoked so sessions created before the reset
// stop working.
func (s *AuthService) SetPassword(userID int64, newPassword string) error {
	user, err := s.UserRepo.FindByID(userID)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("gagal mengenkripsi password baru")
	}
	user.Password = string(hashedPassword)
	if err := s.UserRepo.Update(user); err != nil {
		return fmt.Errorf("AuthService.SetPassword update: %w", err)
	}
	s.revokeAllSessions(userID)
	return nil
}

// RevokeAllSessions invalidates every server-side session of a user.
// It is a no-op when no session repository is wired (backward compat).
func (s *AuthService) RevokeAllSessions(userID int64) {
	s.revokeAllSessions(userID)
}

func (s *AuthService) revokeAllSessions(userID int64) {
	if s.SessionRepo == nil {
		return
	}
	if err := s.SessionRepo.RevokeAllForUser(userID); err != nil {
		log.Printf("AuthService.revokeAllSessions user %d: %v", userID, err)
	}
}
