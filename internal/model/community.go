package model

import "time"

type InvestmentIdea struct {
	ID            int64     `json:"id" db:"id"`
	UserID        int64     `json:"user_id" db:"user_id"`
	Title         string    `json:"title" db:"title"`
	Content       string    `json:"content" db:"content"`
	StockCode     string    `json:"stock_code" db:"stock_code"`
	ChartImage    string    `json:"chart_image" db:"chart_image"`
	IsPublished   bool      `json:"is_published" db:"is_published"`
	LikesCount    int       `json:"likes_count" db:"likes_count"`
	CommentsCount int       `json:"comments_count" db:"comments_count"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	AuthorName    string    `json:"author_name,omitempty" db:"-"`
	StockName     string    `json:"stock_name,omitempty" db:"-"`
	IsLiked       bool      `json:"is_liked,omitempty" db:"-"`
}

type IdeaComment struct {
	ID         int64     `json:"id" db:"id"`
	IdeaID     int64     `json:"idea_id" db:"idea_id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	AuthorName string    `json:"author_name,omitempty" db:"-"`
}

type IdeaLike struct {
	ID        int64     `json:"id" db:"id"`
	IdeaID    int64     `json:"idea_id" db:"idea_id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type StockDiscussion struct {
	ID         int64     `json:"id" db:"id"`
	StockCode  string    `json:"stock_code" db:"stock_code"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	AuthorName string    `json:"author_name,omitempty" db:"-"`
}

type SharedPortfolio struct {
	ID          int64     `json:"id" db:"id"`
	PortfolioID int64     `json:"portfolio_id" db:"portfolio_id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	ShareToken  string    `json:"share_token" db:"share_token"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	ViewsCount  int       `json:"views_count" db:"views_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type LeaderboardEntry struct {
	Rank       int     `json:"rank"`
	UserName   string  `json:"user_name"`
	Return     float64 `json:"return"`
	IdeasCount int     `json:"ideas_count"`
	LikesCount int     `json:"likes_count"`
}
