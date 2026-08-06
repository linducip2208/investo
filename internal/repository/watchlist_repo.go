package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type WatchlistRepository struct {
	DB *sqlx.DB
}

func (r *WatchlistRepository) Create(w *model.Watchlist) (int64, error) {
	query := `INSERT INTO watchlists (user_id, name, is_default) VALUES (:user_id, :name, :is_default)`
	result, err := r.DB.NamedExec(query, w)
	if err != nil {
		return 0, fmt.Errorf("WatchlistRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("WatchlistRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *WatchlistRepository) FindByID(id int64) (*model.Watchlist, error) {
	var watchlist model.Watchlist
	query := `SELECT * FROM watchlists WHERE id = ?`
	if err := r.DB.Get(&watchlist, query, id); err != nil {
		return nil, fmt.Errorf("WatchlistRepository.FindByID: %w", err)
	}
	return &watchlist, nil
}

func (r *WatchlistRepository) FindByUserID(userID int64) ([]model.Watchlist, error) {
	var watchlists []model.Watchlist
	query := `SELECT * FROM watchlists WHERE user_id = ? ORDER BY is_default DESC, created_at DESC`
	if err := r.DB.Select(&watchlists, query, userID); err != nil {
		return nil, fmt.Errorf("WatchlistRepository.FindByUserID: %w", err)
	}
	return watchlists, nil
}

func (r *WatchlistRepository) Update(w *model.Watchlist) error {
	query := `UPDATE watchlists SET user_id = :user_id, name = :name, is_default = :is_default WHERE id = :id`
	_, err := r.DB.NamedExec(query, w)
	if err != nil {
		return fmt.Errorf("WatchlistRepository.Update: %w", err)
	}
	return nil
}

func (r *WatchlistRepository) Delete(id int64) error {
	query := `DELETE FROM watchlists WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("WatchlistRepository.Delete: %w", err)
	}
	return nil
}

type WatchlistItemRepository struct {
	DB *sqlx.DB
}

func (r *WatchlistItemRepository) Add(item *model.WatchlistItem) (int64, error) {
	query := `INSERT INTO watchlist_items (watchlist_id, stock_id) VALUES (:watchlist_id, :stock_id)`
	result, err := r.DB.NamedExec(query, item)
	if err != nil {
		return 0, fmt.Errorf("WatchlistItemRepository.Add: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("WatchlistItemRepository.Add LastInsertId: %w", err)
	}
	return id, nil
}

func (r *WatchlistItemRepository) Remove(watchlistID, stockID int64) error {
	query := `DELETE FROM watchlist_items WHERE watchlist_id = ? AND stock_id = ?`
	_, err := r.DB.Exec(query, watchlistID, stockID)
	if err != nil {
		return fmt.Errorf("WatchlistItemRepository.Remove: %w", err)
	}
	return nil
}

func (r *WatchlistItemRepository) FindByWatchlistID(watchlistID int64) ([]model.WatchlistItem, error) {
	var items []model.WatchlistItem
	query := `SELECT * FROM watchlist_items WHERE watchlist_id = ? ORDER BY added_at DESC`
	if err := r.DB.Select(&items, query, watchlistID); err != nil {
		return nil, fmt.Errorf("WatchlistItemRepository.FindByWatchlistID: %w", err)
	}
	return items, nil
}
