package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type SectorRepository struct {
	DB *sqlx.DB
}

func (r *SectorRepository) FindAll() ([]model.Sector, error) {
	var sectors []model.Sector
	query := `SELECT * FROM sectors ORDER BY name ASC`
	if err := r.DB.Select(&sectors, query); err != nil {
		return nil, fmt.Errorf("SectorRepository.FindAll: %w", err)
	}
	return sectors, nil
}

func (r *SectorRepository) FindBySlug(slug string) (*model.Sector, error) {
	var sector model.Sector
	query := `SELECT * FROM sectors WHERE slug = ?`
	if err := r.DB.Get(&sector, query, slug); err != nil {
		return nil, fmt.Errorf("SectorRepository.FindBySlug: %w", err)
	}
	return &sector, nil
}

func (r *SectorRepository) FindByID(id int64) (*model.Sector, error) {
	var sector model.Sector
	query := `SELECT * FROM sectors WHERE id = ?`
	if err := r.DB.Get(&sector, query, id); err != nil {
		return nil, fmt.Errorf("SectorRepository.FindByID: %w", err)
	}
	return &sector, nil
}
