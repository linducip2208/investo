package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type SettingRepository struct {
	DB *sqlx.DB
}

func (r *SettingRepository) Get(key string) (string, error) {
	var value string
	query := `SELECT ` + "`value`" + ` FROM settings WHERE ` + "`key`" + ` = ?`
	if err := r.DB.Get(&value, query, key); err != nil {
		return "", fmt.Errorf("SettingRepository.Get: %w", err)
	}
	return value, nil
}

func (r *SettingRepository) Set(key, value string) error {
	query := `INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `)`
	_, err := r.DB.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("SettingRepository.Set: %w", err)
	}
	return nil
}

func (r *SettingRepository) GetAll() (map[string]string, error) {
	query := `SELECT ` + "`key`" + `, ` + "`value`" + ` FROM settings`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("SettingRepository.GetAll: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("SettingRepository.GetAll scan: %w", err)
		}
		settings[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("SettingRepository.GetAll rows: %w", err)
	}
	return settings, nil
}
