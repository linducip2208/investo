package repository

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"investo/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type CommunityRepository struct {
	DB *sqlx.DB
}

func (r *CommunityRepository) CreateIdea(idea *model.InvestmentIdea) (int64, error) {
	query := `INSERT INTO investment_ideas (user_id, title, content, stock_code, chart_image, is_published, likes_count, comments_count)
		VALUES (:user_id, :title, :content, :stock_code, :chart_image, :is_published, :likes_count, :comments_count)`
	result, err := r.DB.NamedExec(query, idea)
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateIdea: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateIdea LastInsertId: %w", err)
	}
	return id, nil
}

func (r *CommunityRepository) FindIdeaByID(id int64) (*model.InvestmentIdea, error) {
	type ideaJoin struct {
		model.InvestmentIdea
		AuthorName string `db:"author_name"`
	}
	var row ideaJoin
	query := `SELECT i.*, u.name AS author_name
		FROM investment_ideas i
		JOIN users u ON i.user_id = u.id
		WHERE i.id = ?`
	if err := r.DB.Get(&row, query, id); err != nil {
		return nil, fmt.Errorf("CommunityRepository.FindIdeaByID: %w", err)
	}
	idea := row.InvestmentIdea
	idea.AuthorName = row.AuthorName
	return &idea, nil
}

func (r *CommunityRepository) ListIdeas(offset, limit int, sortBy string, stockCode string) ([]model.InvestmentIdea, int, error) {
	var whereClause string
	var args []interface{}

	if stockCode != "" {
		whereClause = "WHERE i.stock_code = ?"
		args = append(args, stockCode)
	}

	var orderBy string
	switch sortBy {
	case "popular":
		orderBy = "i.likes_count DESC, i.created_at DESC"
	case "trending":
		orderBy = "(i.likes_count + i.comments_count) DESC, i.created_at DESC"
	default:
		orderBy = "i.created_at DESC"
	}

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM investment_ideas i %s`, whereClause)
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("CommunityRepository.ListIdeas count: %w", err)
	}

	var ideas []model.InvestmentIdea
	type ideaJoin struct {
		model.InvestmentIdea
		AuthorName string `db:"author_name"`
	}
	var rows []ideaJoin
	dataQuery := fmt.Sprintf(`SELECT i.*, u.name AS author_name
		FROM investment_ideas i
		JOIN users u ON i.user_id = u.id
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	dataArgs := append(args, limit, offset)
	if err := r.DB.Select(&rows, dataQuery, dataArgs...); err != nil {
		return nil, 0, fmt.Errorf("CommunityRepository.ListIdeas: %w", err)
	}

	ideas = make([]model.InvestmentIdea, len(rows))
	for i, row := range rows {
		ideas[i] = row.InvestmentIdea
		ideas[i].AuthorName = row.AuthorName
	}
	return ideas, total, nil
}

func (r *CommunityRepository) UpdateIdea(idea *model.InvestmentIdea) error {
	query := `UPDATE investment_ideas SET title = :title, content = :content, stock_code = :stock_code,
		chart_image = :chart_image, is_published = :is_published WHERE id = :id`
	_, err := r.DB.NamedExec(query, idea)
	if err != nil {
		return fmt.Errorf("CommunityRepository.UpdateIdea: %w", err)
	}
	return nil
}

func (r *CommunityRepository) DeleteIdea(id int64, userID int64) error {
	query := `DELETE FROM investment_ideas WHERE id = ? AND user_id = ?`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.DeleteIdea: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("idea not found or not owned by user")
	}
	return nil
}

func (r *CommunityRepository) IncrementLikesCount(ideaID int64) error {
	query := `UPDATE investment_ideas SET likes_count = likes_count + 1 WHERE id = ?`
	_, err := r.DB.Exec(query, ideaID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.IncrementLikesCount: %w", err)
	}
	return nil
}

func (r *CommunityRepository) DecrementLikesCount(ideaID int64) error {
	query := `UPDATE investment_ideas SET likes_count = GREATEST(likes_count - 1, 0) WHERE id = ?`
	_, err := r.DB.Exec(query, ideaID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.DecrementLikesCount: %w", err)
	}
	return nil
}

func (r *CommunityRepository) IncrementCommentsCount(ideaID int64) error {
	query := `UPDATE investment_ideas SET comments_count = comments_count + 1 WHERE id = ?`
	_, err := r.DB.Exec(query, ideaID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.IncrementCommentsCount: %w", err)
	}
	return nil
}

func (r *CommunityRepository) CreateComment(comment *model.IdeaComment) (int64, error) {
	query := `INSERT INTO idea_comments (idea_id, user_id, content) VALUES (:idea_id, :user_id, :content)`
	result, err := r.DB.NamedExec(query, comment)
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateComment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateComment LastInsertId: %w", err)
	}
	return id, nil
}

func (r *CommunityRepository) FindCommentsByIdeaID(ideaID int64) ([]model.IdeaComment, error) {
	type commentJoin struct {
		model.IdeaComment
		AuthorName string `db:"author_name"`
	}
	var rows []commentJoin
	query := `SELECT ic.id, ic.idea_id, ic.user_id, ic.content, ic.created_at, u.name AS author_name
		FROM idea_comments ic
		JOIN users u ON ic.user_id = u.id
		WHERE ic.idea_id = ?
		ORDER BY ic.created_at ASC`
	if err := r.DB.Select(&rows, query, ideaID); err != nil {
		return nil, fmt.Errorf("CommunityRepository.FindCommentsByIdeaID: %w", err)
	}
	comments := make([]model.IdeaComment, len(rows))
	for i, row := range rows {
		comments[i] = row.IdeaComment
		comments[i].AuthorName = row.AuthorName
	}
	return comments, nil
}

func (r *CommunityRepository) DeleteComment(id int64, userID int64) error {
	query := `DELETE FROM idea_comments WHERE id = ? AND user_id = ?`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.DeleteComment: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("comment not found or not owned by user")
	}
	return nil
}

func (r *CommunityRepository) LikeIdea(ideaID, userID int64) (bool, error) {
	insertQuery := `INSERT IGNORE INTO idea_likes (idea_id, user_id) VALUES (?, ?)`
	result, err := r.DB.Exec(insertQuery, ideaID, userID)
	if err != nil {
		return false, fmt.Errorf("CommunityRepository.LikeIdea insert: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		if err := r.IncrementLikesCount(ideaID); err != nil {
			return false, err
		}
		return true, nil
	}

	deleteQuery := `DELETE FROM idea_likes WHERE idea_id = ? AND user_id = ?`
	_, err = r.DB.Exec(deleteQuery, ideaID, userID)
	if err != nil {
		return false, fmt.Errorf("CommunityRepository.LikeIdea delete: %w", err)
	}
	if err := r.DecrementLikesCount(ideaID); err != nil {
		return false, err
	}
	return false, nil
}

func (r *CommunityRepository) IsLiked(ideaID, userID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM idea_likes WHERE idea_id = ? AND user_id = ?`
	if err := r.DB.Get(&count, query, ideaID, userID); err != nil {
		return false, fmt.Errorf("CommunityRepository.IsLiked: %w", err)
	}
	return count > 0, nil
}

func (r *CommunityRepository) CreateDiscussion(msg *model.StockDiscussion) (int64, error) {
	query := `INSERT INTO stock_discussions (stock_code, user_id, content) VALUES (:stock_code, :user_id, :content)`
	result, err := r.DB.NamedExec(query, msg)
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateDiscussion: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateDiscussion LastInsertId: %w", err)
	}
	return id, nil
}

func (r *CommunityRepository) FindDiscussionsByStock(stockCode string, offset, limit int) ([]model.StockDiscussion, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM stock_discussions WHERE stock_code = ?`
	if err := r.DB.Get(&total, countQuery, stockCode); err != nil {
		return nil, 0, fmt.Errorf("CommunityRepository.FindDiscussionsByStock count: %w", err)
	}

	type discussionJoin struct {
		model.StockDiscussion
		AuthorName string `db:"author_name"`
	}
	var rows []discussionJoin
	query := `SELECT sd.id, sd.stock_code, sd.user_id, sd.content, sd.created_at, u.name AS author_name
		FROM stock_discussions sd
		JOIN users u ON sd.user_id = u.id
		WHERE sd.stock_code = ?
		ORDER BY sd.created_at DESC
		LIMIT ? OFFSET ?`
	if err := r.DB.Select(&rows, query, stockCode, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("CommunityRepository.FindDiscussionsByStock: %w", err)
	}

	discussions := make([]model.StockDiscussion, len(rows))
	for i, row := range rows {
		discussions[i] = row.StockDiscussion
		discussions[i].AuthorName = row.AuthorName
	}

	return discussions, total, nil
}

func (r *CommunityRepository) CreateShare(share *model.SharedPortfolio) (int64, error) {
	if share.ShareToken == "" {
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return 0, fmt.Errorf("CommunityRepository.CreateShare rand: %w", err)
		}
		share.ShareToken = hex.EncodeToString(tokenBytes)
	}

	query := `INSERT INTO portfolios_shared (portfolio_id, user_id, share_token, is_active, views_count)
		VALUES (:portfolio_id, :user_id, :share_token, :is_active, :views_count)`
	result, err := r.DB.NamedExec(query, share)
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateShare: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("CommunityRepository.CreateShare LastInsertId: %w", err)
	}
	return id, nil
}

func (r *CommunityRepository) FindShareByToken(token string) (*model.SharedPortfolio, error) {
	var share model.SharedPortfolio
	query := `SELECT * FROM portfolios_shared WHERE share_token = ? AND is_active = TRUE`
	if err := r.DB.Get(&share, query, token); err != nil {
		return nil, fmt.Errorf("CommunityRepository.FindShareByToken: %w", err)
	}
	return &share, nil
}

func (r *CommunityRepository) IncrementShareViews(shareID int64) error {
	query := `UPDATE portfolios_shared SET views_count = views_count + 1 WHERE id = ?`
	_, err := r.DB.Exec(query, shareID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.IncrementShareViews: %w", err)
	}
	return nil
}

func (r *CommunityRepository) ListSharesByUser(userID int64) ([]model.SharedPortfolio, error) {
	var shares []model.SharedPortfolio
	query := `SELECT * FROM portfolios_shared WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&shares, query, userID); err != nil {
		return nil, fmt.Errorf("CommunityRepository.ListSharesByUser: %w", err)
	}
	return shares, nil
}

func (r *CommunityRepository) DeactivateShare(id int64, userID int64) error {
	query := `UPDATE portfolios_shared SET is_active = FALSE WHERE id = ? AND user_id = ?`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("CommunityRepository.DeactivateShare: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("share not found or not owned by user")
	}
	return nil
}

func (r *CommunityRepository) GetLeaderboard(limit int) ([]model.LeaderboardEntry, error) {
	var entries []model.LeaderboardEntry
	query := `SELECT u.name AS user_name,
		COALESCE(SUM(i.likes_count), 0) AS likes_count,
		COUNT(DISTINCT i.id) AS ideas_count
		FROM users u
		LEFT JOIN investment_ideas i ON i.user_id = u.id AND i.is_published = TRUE
		GROUP BY u.id, u.name
		ORDER BY likes_count DESC, ideas_count DESC
		LIMIT ?`

	if err := r.DB.Select(&entries, query, limit); err != nil {
		return []model.LeaderboardEntry{}, nil
	}

	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries, nil
}

func (r *CommunityRepository) GetIdeasByUserID(userID int64) ([]model.InvestmentIdea, error) {
	var ideas []model.InvestmentIdea
	query := `SELECT * FROM investment_ideas WHERE user_id = ? ORDER BY created_at DESC`
	if err := r.DB.Select(&ideas, query, userID); err != nil {
		return nil, fmt.Errorf("CommunityRepository.GetIdeasByUserID: %w", err)
	}
	return ideas, nil
}

func (r *CommunityRepository) FindCommentByID(id int64) (*model.IdeaComment, error) {
	var comment model.IdeaComment
	query := `SELECT * FROM idea_comments WHERE id = ?`
	if err := r.DB.Get(&comment, query, id); err != nil {
		return nil, fmt.Errorf("CommunityRepository.FindCommentByID: %w", err)
	}
	return &comment, nil
}

func (r *CommunityRepository) IdeaCountByUserID(userID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM investment_ideas WHERE user_id = ?`
	if err := r.DB.Get(&count, query, userID); err != nil {
		return 0, fmt.Errorf("CommunityRepository.IdeaCountByUserID: %w", err)
	}
	return count, nil
}

func (r *CommunityRepository) TotalLikesByUserID(userID int64) (int, error) {
	var count int
	query := `SELECT COALESCE(SUM(likes_count), 0) FROM investment_ideas WHERE user_id = ?`
	if err := r.DB.Get(&count, query, userID); err != nil {
		return 0, fmt.Errorf("CommunityRepository.TotalLikesByUserID: %w", err)
	}
	return count, nil
}

func (r *CommunityRepository) FindShareByPortfolioID(portfolioID int64, userID int64) (*model.SharedPortfolio, error) {
	var share model.SharedPortfolio
	query := `SELECT * FROM portfolios_shared WHERE portfolio_id = ? AND user_id = ? AND is_active = TRUE`
	if err := r.DB.Get(&share, query, portfolioID, userID); err != nil {
		return nil, fmt.Errorf("CommunityRepository.FindShareByPortfolioID: %w", err)
	}
	return &share, nil
}

func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "baru saja"
	case duration < time.Hour:
		m := int(duration.Minutes())
		if m == 1 {
			return "1 menit lalu"
		}
		return fmt.Sprintf("%d menit lalu", m)
	case duration < 24*time.Hour:
		h := int(duration.Hours())
		if h == 1 {
			return "1 jam lalu"
		}
		return fmt.Sprintf("%d jam lalu", h)
	case duration < 7*24*time.Hour:
		d := int(duration.Hours() / 24)
		if d == 1 {
			return "1 hari lalu"
		}
		return fmt.Sprintf("%d hari lalu", d)
	case duration < 30*24*time.Hour:
		w := int(duration.Hours() / 24 / 7)
		if w == 1 {
			return "1 minggu lalu"
		}
		return fmt.Sprintf("%d minggu lalu", w)
	case duration < 365*24*time.Hour:
		m := int(duration.Hours() / 24 / 30)
		if m == 1 {
			return "1 bulan lalu"
		}
		return fmt.Sprintf("%d bulan lalu", m)
	default:
		y := int(duration.Hours() / 24 / 365)
		if y == 1 {
			return "1 tahun lalu"
		}
		return fmt.Sprintf("%d tahun lalu", y)
	}
}
