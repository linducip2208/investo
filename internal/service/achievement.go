package service

import (
	"investo/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type AchievementService struct {
	DB *sqlx.DB
}

func NewAchievementService(db *sqlx.DB) *AchievementService {
	return &AchievementService{DB: db}
}

func (s *AchievementService) CheckAchievements(userID int64) ([]model.Achievement, error) {
	rows, err := s.getUserAchievementRows(userID)
	if err != nil {
		return nil, err
	}

	achievementMap := make(map[string]*model.AchievementRow)
	for _, r := range rows {
		rr := r
		achievementMap[r.AchievementKey] = &rr
	}

	s.autoCheck(userID, achievementMap)

	rows, _ = s.getUserAchievementRows(userID)
	achievementMap = make(map[string]*model.AchievementRow)
	for _, r := range rows {
		rr := r
		achievementMap[r.AchievementKey] = &rr
	}

	unlockedCount := 0
	for _, r := range rows {
		if r.UnlockedAt != nil {
			unlockedCount++
		}
	}

	if unlockedCount >= 9 {
		s.unlockAchievement(userID, "investo_master")
	}

	rows, _ = s.getUserAchievementRows(userID)
	achievementMap = make(map[string]*model.AchievementRow)
	for _, r := range rows {
		rr := r
		achievementMap[r.AchievementKey] = &rr
	}

	var achievements []model.Achievement
	for _, prebuilt := range model.PrebuiltAchievements {
		row, exists := achievementMap[prebuilt.Key]
		ach := prebuilt
		if exists && row.UnlockedAt != nil {
			ach.Unlocked = true
			ach.Progress = row.Progress
			ach.UnlockedAt = row.UnlockedAt
		} else if exists {
			ach.Progress = row.Progress
		}
		ach.Target = prebuilt.Target
		achievements = append(achievements, ach)
	}

	return achievements, nil
}

func (s *AchievementService) GetUserAchievements(userID int64) ([]model.Achievement, error) {
	achievements, err := s.CheckAchievements(userID)
	if err != nil {
		return nil, err
	}

	var unlocked []model.Achievement
	for _, a := range achievements {
		if a.Unlocked {
			unlocked = append(unlocked, a)
		}
	}
	if unlocked == nil {
		unlocked = []model.Achievement{}
	}
	return unlocked, nil
}

func (s *AchievementService) autoCheck(userID int64, existing map[string]*model.AchievementRow) {
	if _, ok := existing["newbie_investor"]; !ok || existing["newbie_investor"].UnlockedAt == nil {
		var portfolioCount int
		s.DB.Get(&portfolioCount, "SELECT COUNT(*) FROM portfolios WHERE user_id = ?", userID)
		if portfolioCount > 0 {
			s.unlockAchievement(userID, "newbie_investor")
		}
	}

	if _, ok := existing["first_trade"]; !ok || existing["first_trade"].UnlockedAt == nil {
		var itemCount int
		s.DB.Get(&itemCount, `SELECT COUNT(*) FROM portfolio_items pi JOIN portfolios p ON pi.portfolio_id = p.id WHERE p.user_id = ?`, userID)
		if itemCount > 0 {
			s.unlockAchievement(userID, "first_trade")
		}
	}

	if _, ok := existing["analyst"]; !ok || (existing["analyst"].UnlockedAt == nil && existing["analyst"].Progress < 10) {
		var screenerCount int
		s.DB.Get(&screenerCount, "SELECT COUNT(*) FROM saved_screeners WHERE user_id = ?", userID)
		if screenerCount >= 10 {
			s.unlockAchievement(userID, "analyst")
		} else if _, ok := existing["analyst"]; !ok {
			s.DB.Exec(`INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress) VALUES (?, 'analyst', ?)`, userID, screenerCount)
		} else {
			s.DB.Exec(`UPDATE user_achievements SET progress = ? WHERE user_id = ? AND achievement_key = 'analyst'`, screenerCount, userID)
		}
	}

	if _, ok := existing["idea_maker"]; !ok || (existing["idea_maker"].UnlockedAt == nil && existing["idea_maker"].Progress < 5) {
		var ideaCount int
		s.DB.Get(&ideaCount, "SELECT COUNT(*) FROM community_ideas WHERE user_id = ?", userID)
		if ideaCount >= 5 {
			s.unlockAchievement(userID, "idea_maker")
		} else if _, ok := existing["idea_maker"]; !ok {
			s.DB.Exec(`INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress) VALUES (?, 'idea_maker', ?)`, userID, ideaCount)
		} else {
			s.DB.Exec(`UPDATE user_achievements SET progress = ? WHERE user_id = ? AND achievement_key = 'idea_maker'`, ideaCount, userID)
		}
	}

	if _, ok := existing["7day_streak"]; !ok || existing["7day_streak"].UnlockedAt == nil {
		var streak int
		s.DB.Get(&streak, "SELECT COALESCE(MAX(login_streak), 0) FROM user_login_streaks WHERE user_id = ?", userID)
		if streak >= 7 {
			s.unlockAchievement(userID, "7day_streak")
		} else if _, ok := existing["7day_streak"]; !ok {
			s.DB.Exec(`INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress) VALUES (?, '7day_streak', ?)`, userID, streak)
		} else {
			s.DB.Exec(`UPDATE user_achievements SET progress = ? WHERE user_id = ? AND achievement_key = '7day_streak'`, streak, userID)
		}
	}

	if _, ok := existing["dividend_king"]; !ok || (existing["dividend_king"].UnlockedAt == nil && existing["dividend_king"].Progress < 10) {
		var divCount int
		s.DB.Get(&divCount, `SELECT COUNT(*) FROM stock_actions sa JOIN portfolio_items pi ON sa.stock_id = pi.stock_id JOIN portfolios p ON pi.portfolio_id = p.id WHERE p.user_id = ? AND sa.action_type = 'dividend'`, userID)
		if divCount >= 10 {
			s.unlockAchievement(userID, "dividend_king")
		} else if _, ok := existing["dividend_king"]; !ok {
			s.DB.Exec(`INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress) VALUES (?, 'dividend_king', ?)`, userID, divCount)
		} else {
			s.DB.Exec(`UPDATE user_achievements SET progress = ? WHERE user_id = ? AND achievement_key = 'dividend_king'`, divCount, userID)
		}
	}

	if _, ok := existing["knowledge_seeker"]; !ok || (existing["knowledge_seeker"].UnlockedAt == nil && existing["knowledge_seeker"].Progress < 20) {
		var blogReadCount int
		s.DB.Get(&blogReadCount, `SELECT COUNT(*) FROM blog_reads WHERE user_id = ?`, userID)
		if blogReadCount >= 20 {
			s.unlockAchievement(userID, "knowledge_seeker")
		} else if _, ok := existing["knowledge_seeker"]; !ok {
			s.DB.Exec(`INSERT IGNORE INTO user_achievements (user_id, achievement_key, progress) VALUES (?, 'knowledge_seeker', ?)`, userID, blogReadCount)
		} else {
			s.DB.Exec(`UPDATE user_achievements SET progress = ? WHERE user_id = ? AND achievement_key = 'knowledge_seeker'`, blogReadCount, userID)
		}
	}

	_ = existing
}

func (s *AchievementService) getUserAchievementRows(userID int64) ([]model.AchievementRow, error) {
	var rows []model.AchievementRow
	query := `SELECT * FROM user_achievements WHERE user_id = ?`
	if err := s.DB.Select(&rows, query, userID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *AchievementService) unlockAchievement(userID int64, key string) {
	now := time.Now()
	query := `INSERT INTO user_achievements (user_id, achievement_key, progress, unlocked_at)
		VALUES (?, ?, 1, ?)
		ON DUPLICATE KEY UPDATE progress = 1, unlocked_at = ?`
	s.DB.Exec(query, userID, key, now, now)
}
