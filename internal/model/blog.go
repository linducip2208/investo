package model

import "time"

type BlogPost struct {
	ID              int64     `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	Slug            string    `json:"slug" db:"slug"`
	Content         string    `json:"content" db:"content"`
	Excerpt         string    `json:"excerpt" db:"excerpt"`
	FeaturedImage   string    `json:"featured_image" db:"featured_image"`
	CategoryID      int64     `json:"category_id" db:"category_id"`
	AuthorID        int64     `json:"author_id" db:"author_id"`
	PublishedAt     time.Time `json:"published_at" db:"published_at"`
	IsPublished     bool      `json:"is_published" db:"is_published"`
	MetaTitle       string    `json:"meta_title" db:"meta_title"`
	MetaDescription string    `json:"meta_description" db:"meta_description"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type BlogCategory struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	Description string `json:"description" db:"description"`
}
