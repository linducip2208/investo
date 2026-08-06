package service

import (
	"time"

	"github.com/jmoiron/sqlx"
)

type AdminStats struct {
	TotalUsers          int             `json:"total_users"`
	ActiveToday         int             `json:"active_today"`
	NewThisWeek         int             `json:"new_this_week"`
	NewThisMonth        int             `json:"new_this_month"`
	TotalPortfolios     int             `json:"total_portfolios"`
	TotalWatchlists     int             `json:"total_watchlists"`
	TotalAlerts         int             `json:"total_alerts"`
	TotalIdeas          int             `json:"total_ideas"`
	TotalComments       int             `json:"total_comments"`
	TotalLikes          int             `json:"total_likes"`
	TotalStocks         int             `json:"total_stocks"`
	TotalNews           int             `json:"total_news"`
	AvgSessionDuration  float64         `json:"avg_session_duration"`
	UserGrowth          []UserGrowthDay `json:"user_growth"`
}

type UserGrowthDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type UserWithStats struct {
	ID            int64     `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Email         string    `json:"email" db:"email"`
	Role          string    `json:"role" db:"role"`
	Plan          string    `json:"plan" db:"plan"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	PortfolioCount int      `json:"portfolio_count" db:"portfolio_count"`
	WatchlistCount int      `json:"watchlist_count" db:"watchlist_count"`
	LoginCount    int       `json:"login_count" db:"login_count"`
	LastActive    string    `json:"last_active" db:"last_active"`
}

type PageFeatureUsage struct {
	Page  string `json:"page" db:"page"`
	Views int    `json:"views" db:"views"`
}

type AdminAnalyticsService struct {
	DB *sqlx.DB
}

func (s *AdminAnalyticsService) GetDashboardStats() (*AdminStats, error) {
	stats := &AdminStats{}

	s.DB.Get(&stats.TotalUsers, "SELECT COUNT(*) FROM users")

	now := time.Now()

	today := now.Format("2006-01-02")
	s.DB.Get(&stats.ActiveToday, "SELECT COUNT(DISTINCT user_id) FROM user_sessions WHERE DATE(last_active) = ?", today)

	weekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
	s.DB.Get(&stats.NewThisWeek, "SELECT COUNT(*) FROM users WHERE DATE(created_at) >= ?", weekAgo)

	monthAgo := now.AddDate(0, -1, 0).Format("2006-01-02")
	s.DB.Get(&stats.NewThisMonth, "SELECT COUNT(*) FROM users WHERE DATE(created_at) >= ?", monthAgo)

	s.DB.Get(&stats.TotalPortfolios, "SELECT COUNT(*) FROM portfolios")
	s.DB.Get(&stats.TotalWatchlists, "SELECT COUNT(*) FROM watchlists")
	s.DB.Get(&stats.TotalAlerts, "SELECT COUNT(*) FROM alerts")
	s.DB.Get(&stats.TotalIdeas, "SELECT COUNT(*) FROM community_ideas")
	s.DB.Get(&stats.TotalComments, "SELECT COUNT(*) FROM community_comments")
	s.DB.Get(&stats.TotalLikes, "SELECT COUNT(*) FROM community_likes")
	s.DB.Get(&stats.TotalStocks, "SELECT COUNT(*) FROM stocks")
	s.DB.Get(&stats.TotalNews, "SELECT COUNT(*) FROM news")

	var avgSecs float64
	s.DB.Get(&avgSecs, "SELECT COALESCE(AVG(duration_seconds), 0) FROM user_sessions WHERE duration_seconds > 0")
	stats.AvgSessionDuration = avgSecs

	growth, _ := s.getUserGrowth(30)
	stats.UserGrowth = growth

	return stats, nil
}

func (s *AdminAnalyticsService) getUserGrowth(days int) ([]UserGrowthDay, error) {
	var rows []UserGrowthDay
	query := `SELECT DATE(created_at) as date, COUNT(*) as count FROM users
		WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY DATE(created_at) ORDER BY date ASC`
	err := s.DB.Select(&rows, query, days)
	if err != nil {
		return []UserGrowthDay{}, nil
	}
	if rows == nil {
		rows = []UserGrowthDay{}
	}
	return rows, nil
}

func (s *AdminAnalyticsService) GetUserList(page, limit int, search string) ([]UserWithStats, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	offset := (page - 1) * limit

	var total int
	countQ := "SELECT COUNT(*) FROM users"
	args := []interface{}{}
	if search != "" {
		countQ += " WHERE name LIKE ? OR email LIKE ?"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	s.DB.Get(&total, countQ, args...)

	query := `SELECT u.id, u.name, u.email, u.role, COALESCE(u.plan, 'free') as plan, u.created_at,
		(SELECT COUNT(*) FROM portfolios WHERE user_id = u.id) as portfolio_count,
		(SELECT COUNT(*) FROM watchlists WHERE user_id = u.id) as watchlist_count,
		0 as login_count,
		'' as last_active
		FROM users u`
	if search != "" {
		query += " WHERE u.name LIKE ? OR u.email LIKE ?"
	} else {
		query += " WHERE 1=1"
	}
	query += " ORDER BY u.id DESC LIMIT ? OFFSET ?"

	var allArgs []interface{}
	if search != "" {
		allArgs = append(allArgs, "%"+search+"%", "%"+search+"%")
	}
	allArgs = append(allArgs, limit, offset)

	var users []UserWithStats
	err := s.DB.Select(&users, query, allArgs...)
	if err != nil {
		return nil, 0, err
	}
	if users == nil {
		users = []UserWithStats{}
	}

	for i := range users {
		var loginCount int
		s.DB.Get(&loginCount, "SELECT COUNT(*) FROM user_sessions WHERE user_id = ?", users[i].ID)
		users[i].LoginCount = loginCount

		var lastActive *time.Time
		s.DB.Get(&lastActive, "SELECT MAX(last_active) FROM user_sessions WHERE user_id = ?", users[i].ID)
		if lastActive != nil {
			users[i].LastActive = lastActive.Format("2006-01-02 15:04")
		} else {
			users[i].LastActive = "-"
		}
	}

	return users, total, nil
}

func (s *AdminAnalyticsService) GetFeatureUsage() ([]PageFeatureUsage, error) {
	var rows []PageFeatureUsage
	err := s.DB.Select(&rows, `SELECT page, COUNT(*) as views FROM page_views
		WHERE viewed_at >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
		GROUP BY page ORDER BY views DESC LIMIT 20`)
	if err != nil {
		return []PageFeatureUsage{}, nil
	}
	if rows == nil {
		rows = []PageFeatureUsage{}
	}
	return rows, nil
}

func EnsureSessionTable(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_sessions (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL,
		last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		duration_seconds INT DEFAULT 0,
		INDEX idx_user_sessions_user (user_id),
		INDEX idx_user_sessions_active (last_active)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS page_views (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		page VARCHAR(255) NOT NULL,
		viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_page_views_page (page),
		INDEX idx_page_views_at (viewed_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}
