package service

import (
	"errors"
	"fmt"
	"investo/internal/model"
	"investo/internal/repository"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrEmailExists        = errors.New("email sudah terdaftar")
	ErrSessionNotFound    = errors.New("session tidak ditemukan")
)

type AuthService struct {
	UserRepo *repository.UserRepository
}

func (s *AuthService) Register(name, email, password string) (*model.User, error) {
	existing, err := s.UserRepo.FindByEmail(email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
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

func (s *AuthService) Login(email, password string) (*model.User, error) {
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
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return errors.New("gagal mengenkripsi password baru")
		}
		user.Password = string(hashedPassword)
	}

	user.Name = name
	user.Email = email

	return s.UserRepo.Update(user)
}
