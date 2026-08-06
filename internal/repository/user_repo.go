package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	DB *sqlx.DB
}

func (r *UserRepository) Create(user *model.User) (int64, error) {
	query := `INSERT INTO users (name, email, password, role) VALUES (:name, :email, :password, :role)`
	result, err := r.DB.NamedExec(query, user)
	if err != nil {
		return 0, fmt.Errorf("UserRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("UserRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE id = ?`
	if err := r.DB.Get(&user, query, id); err != nil {
		return nil, fmt.Errorf("UserRepository.FindByID: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	query := `SELECT * FROM users WHERE email = ?`
	if err := r.DB.Get(&user, query, email); err != nil {
		return nil, fmt.Errorf("UserRepository.FindByEmail: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	query := `UPDATE users SET name = :name, email = :email, password = :password, role = :role WHERE id = :id`
	_, err := r.DB.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("UserRepository.Update: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(id int64) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("UserRepository.Delete: %w", err)
	}
	return nil
}

func (r *UserRepository) List(offset, limit int) ([]model.User, int, error) {
	var users []model.User
	query := `SELECT * FROM users ORDER BY id DESC LIMIT ? OFFSET ?`
	if err := r.DB.Select(&users, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List: %w", err)
	}

	total, err := r.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("UserRepository.List count: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) Count() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	if err := r.DB.Get(&count, query); err != nil {
		return 0, fmt.Errorf("UserRepository.Count: %w", err)
	}
	return count, nil
}
