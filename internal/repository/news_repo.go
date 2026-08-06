package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type NewsRepository struct {
	DB *sqlx.DB
}

func (r *NewsRepository) Create(n *model.News) (int64, error) {
	query := `INSERT INTO news (title, slug, content, source, source_url, image_url, published_at, sentiment_score)
		VALUES (:title, :slug, :content, :source, :source_url, :image_url, :published_at, :sentiment_score)`
	result, err := r.DB.NamedExec(query, n)
	if err != nil {
		return 0, fmt.Errorf("NewsRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("NewsRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *NewsRepository) FindByID(id int64) (*model.News, error) {
	var news model.News
	query := `SELECT * FROM news WHERE id = ?`
	if err := r.DB.Get(&news, query, id); err != nil {
		return nil, fmt.Errorf("NewsRepository.FindByID: %w", err)
	}
	return &news, nil
}

func (r *NewsRepository) FindBySlug(slug string) (*model.News, error) {
	var news model.News
	query := `SELECT * FROM news WHERE slug = ?`
	if err := r.DB.Get(&news, query, slug); err != nil {
		return nil, fmt.Errorf("NewsRepository.FindBySlug: %w", err)
	}
	return &news, nil
}

func (r *NewsRepository) List(offset, limit int) ([]model.News, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM news`
	if err := r.DB.Get(&total, countQuery); err != nil {
		return nil, 0, fmt.Errorf("NewsRepository.List count: %w", err)
	}

	var news []model.News
	query := `SELECT * FROM news ORDER BY published_at DESC LIMIT ? OFFSET ?`
	if err := r.DB.Select(&news, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("NewsRepository.List: %w", err)
	}
	return news, total, nil
}

func (r *NewsRepository) FindByStockID(stockID int64, limit int) ([]model.News, error) {
	var news []model.News
	query := `SELECT n.* FROM news n
		INNER JOIN news_stocks ns ON n.id = ns.news_id
		WHERE ns.stock_id = ?
		ORDER BY n.published_at DESC LIMIT ?`
	if err := r.DB.Select(&news, query, stockID, limit); err != nil {
		return nil, fmt.Errorf("NewsRepository.FindByStockID: %w", err)
	}
	return news, nil
}

func (r *NewsRepository) BulkInsert(newsList []model.News) error {
	if len(newsList) == 0 {
		return nil
	}

	query := `INSERT IGNORE INTO news (title, slug, content, source, source_url, image_url, published_at, sentiment_score)
		VALUES (:title, :slug, :content, :source, :source_url, :image_url, :published_at, :sentiment_score)`
	_, err := r.DB.NamedExec(query, newsList)
	if err != nil {
		return fmt.Errorf("NewsRepository.BulkInsert: %w", err)
	}
	return nil
}

func (r *NewsRepository) LinkStock(newsID, stockID int64) error {
	query := `INSERT IGNORE INTO news_stocks (news_id, stock_id) VALUES (?, ?)`
	_, err := r.DB.Exec(query, newsID, stockID)
	if err != nil {
		return fmt.Errorf("NewsRepository.LinkStock: %w", err)
	}
	return nil
}
