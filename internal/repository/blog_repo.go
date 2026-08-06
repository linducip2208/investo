package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type BlogRepository struct {
	DB *sqlx.DB
}

func (r *BlogRepository) CreatePost(p *model.BlogPost) (int64, error) {
	query := `INSERT INTO blog_posts (title, slug, content, excerpt, featured_image, category_id, author_id,
		published_at, is_published, meta_title, meta_description)
		VALUES (:title, :slug, :content, :excerpt, :featured_image, :category_id, :author_id,
		:published_at, :is_published, :meta_title, :meta_description)`
	result, err := r.DB.NamedExec(query, p)
	if err != nil {
		return 0, fmt.Errorf("BlogRepository.CreatePost: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("BlogRepository.CreatePost LastInsertId: %w", err)
	}
	return id, nil
}

func (r *BlogRepository) FindPostByID(id int64) (*model.BlogPost, error) {
	var post model.BlogPost
	query := `SELECT * FROM blog_posts WHERE id = ?`
	if err := r.DB.Get(&post, query, id); err != nil {
		return nil, fmt.Errorf("BlogRepository.FindPostByID: %w", err)
	}
	return &post, nil
}

func (r *BlogRepository) FindPostBySlug(slug string) (*model.BlogPost, error) {
	var post model.BlogPost
	query := `SELECT * FROM blog_posts WHERE slug = ?`
	if err := r.DB.Get(&post, query, slug); err != nil {
		return nil, fmt.Errorf("BlogRepository.FindPostBySlug: %w", err)
	}
	return &post, nil
}

func (r *BlogRepository) ListPosts(offset, limit int, categoryID int64) ([]model.BlogPost, int, error) {
	var posts []model.BlogPost

	where := "WHERE 1=1"
	args := []interface{}{}

	if categoryID > 0 {
		where += " AND category_id = ?"
		args = append(args, categoryID)
	}

	countQuery := `SELECT COUNT(*) FROM blog_posts ` + where
	var total int
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("BlogRepository.ListPosts count: %w", err)
	}

	dataArgs := append(args, limit, offset)
	dataQuery := `SELECT * FROM blog_posts ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	if err := r.DB.Select(&posts, dataQuery, dataArgs...); err != nil {
		return nil, 0, fmt.Errorf("BlogRepository.ListPosts: %w", err)
	}

	return posts, total, nil
}

func (r *BlogRepository) ListPublished(offset, limit int, categoryID int64) ([]model.BlogPost, int, error) {
	var posts []model.BlogPost

	where := "WHERE is_published = true AND published_at <= NOW()"
	args := []interface{}{}

	if categoryID > 0 {
		where += " AND category_id = ?"
		args = append(args, categoryID)
	}

	countQuery := `SELECT COUNT(*) FROM blog_posts ` + where
	var total int
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("BlogRepository.ListPublished count: %w", err)
	}

	dataArgs := append(args, limit, offset)
	dataQuery := `SELECT * FROM blog_posts ` + where + ` ORDER BY published_at DESC LIMIT ? OFFSET ?`
	if err := r.DB.Select(&posts, dataQuery, dataArgs...); err != nil {
		return nil, 0, fmt.Errorf("BlogRepository.ListPublished: %w", err)
	}

	return posts, total, nil
}

func (r *BlogRepository) UpdatePost(p *model.BlogPost) error {
	query := `UPDATE blog_posts SET title = :title, slug = :slug, content = :content, excerpt = :excerpt,
		featured_image = :featured_image, category_id = :category_id, author_id = :author_id,
		published_at = :published_at, is_published = :is_published,
		meta_title = :meta_title, meta_description = :meta_description WHERE id = :id`
	_, err := r.DB.NamedExec(query, p)
	if err != nil {
		return fmt.Errorf("BlogRepository.UpdatePost: %w", err)
	}
	return nil
}

func (r *BlogRepository) DeletePost(id int64) error {
	query := `DELETE FROM blog_posts WHERE id = ?`
	_, err := r.DB.Exec(query, id)
	if err != nil {
		return fmt.Errorf("BlogRepository.DeletePost: %w", err)
	}
	return nil
}

func (r *BlogRepository) FindAllCategories() ([]model.BlogCategory, error) {
	var categories []model.BlogCategory
	query := `SELECT * FROM blog_categories ORDER BY name ASC`
	if err := r.DB.Select(&categories, query); err != nil {
		return nil, fmt.Errorf("BlogRepository.FindAllCategories: %w", err)
	}
	return categories, nil
}

func (r *BlogRepository) CreateCategory(c *model.BlogCategory) (int64, error) {
	query := `INSERT INTO blog_categories (name, slug, description) VALUES (:name, :slug, :description)`
	result, err := r.DB.NamedExec(query, c)
	if err != nil {
		return 0, fmt.Errorf("BlogRepository.CreateCategory: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("BlogRepository.CreateCategory LastInsertId: %w", err)
	}
	return id, nil
}

func (r *BlogRepository) FindCategoryBySlug(slug string) (*model.BlogCategory, error) {
	var category model.BlogCategory
	query := `SELECT * FROM blog_categories WHERE slug = ?`
	if err := r.DB.Get(&category, query, slug); err != nil {
		return nil, fmt.Errorf("BlogRepository.FindCategoryBySlug: %w", err)
	}
	return &category, nil
}
