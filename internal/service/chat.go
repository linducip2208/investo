package service

import (
	"fmt"
	"log"
	"time"

	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type ChatService struct {
	DB *sqlx.DB
}

func NewChatService(db *sqlx.DB) *ChatService {
	return &ChatService{DB: db}
}

func (s *ChatService) SendMessage(senderID int64, receiverID *int64, room *string, message string) (*model.ChatMessage, error) {
	msg := &model.ChatMessage{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Room:       room,
		Message:    message,
	}
	query := `INSERT INTO chat_messages (sender_id, receiver_id, room, message) VALUES (?, ?, ?, ?)`
	result, err := s.DB.Exec(query, senderID, receiverID, room, message)
	if err != nil {
		return nil, fmt.Errorf("ChatService.SendMessage: %w", err)
	}
	id, _ := result.LastInsertId()
	msg.ID = id
	msg.CreatedAt = time.Now()

	var senderName string
	if err := s.DB.Get(&senderName, "SELECT name FROM users WHERE id = ?", senderID); err == nil {
		msg.SenderName = senderName
	}

	return msg, nil
}

func (s *ChatService) GetRoomMessages(room string, limit int) ([]model.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	var messages []model.ChatMessage
	query := `SELECT cm.*, u.name AS sender_name FROM chat_messages cm
		JOIN users u ON cm.sender_id = u.id
		WHERE cm.room = ?
		ORDER BY cm.created_at DESC
		LIMIT ?`
	if err := s.DB.Select(&messages, query, room, limit); err != nil {
		return nil, fmt.Errorf("ChatService.GetRoomMessages: %w", err)
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (s *ChatService) GetDMMessages(user1ID, user2ID int64, limit int) ([]model.ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	var messages []model.ChatMessage
	query := `SELECT cm.*, u.name AS sender_name FROM chat_messages cm
		JOIN users u ON cm.sender_id = u.id
		WHERE cm.room IS NULL AND (
			(cm.sender_id = ? AND cm.receiver_id = ?) OR
			(cm.sender_id = ? AND cm.receiver_id = ?)
		)
		ORDER BY cm.created_at DESC
		LIMIT ?`
	if err := s.DB.Select(&messages, query, user1ID, user2ID, user2ID, user1ID, limit); err != nil {
		return nil, fmt.Errorf("ChatService.GetDMMessages: %w", err)
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (s *ChatService) GetDMConversations(userID int64) ([]model.DMConversation, error) {
	query := `SELECT
		CASE WHEN cm.sender_id = ? THEN cm.receiver_id ELSE cm.sender_id END AS user_id,
		u2.name AS user_name,
		(SELECT cm2.message FROM chat_messages cm2
		 WHERE cm2.room IS NULL AND ((cm2.sender_id = ? AND cm2.receiver_id = u2.id) OR (cm2.sender_id = u2.id AND cm2.receiver_id = ?))
		 ORDER BY cm2.created_at DESC LIMIT 1) AS last_message,
		(SELECT cm2.created_at FROM chat_messages cm2
		 WHERE cm2.room IS NULL AND ((cm2.sender_id = ? AND cm2.receiver_id = u2.id) OR (cm2.sender_id = u2.id AND cm2.receiver_id = ?))
		 ORDER BY cm2.created_at DESC LIMIT 1) AS last_message_at,
		COUNT(DISTINCT cm.id) AS unread_count
		FROM chat_messages cm
		JOIN users u2 ON u2.id = CASE WHEN cm.sender_id = ? THEN cm.receiver_id ELSE cm.sender_id END
		WHERE cm.room IS NULL AND (cm.sender_id = ? OR cm.receiver_id = ?)
		GROUP BY user_id, u2.name
		ORDER BY last_message_at DESC`
	var conversations []model.DMConversation
	uID := userID
	if err := s.DB.Select(&conversations, query, uID, uID, uID, uID, uID, uID, uID, uID); err != nil {
		return nil, fmt.Errorf("ChatService.GetDMConversations: %w", err)
	}
	return conversations, nil
}

func (s *ChatService) GetActiveStockRooms() ([]string, error) {
	var rooms []string
	query := `SELECT DISTINCT room FROM chat_messages WHERE room IS NOT NULL AND created_at > DATE_SUB(NOW(), INTERVAL 7 DAY) ORDER BY room`
	if err := s.DB.Select(&rooms, query); err != nil {
		return nil, fmt.Errorf("ChatService.GetActiveStockRooms: %w", err)
	}
	return rooms, nil
}

func (s *ChatService) Follow(followerID, followingID int64) error {
	query := `INSERT IGNORE INTO user_follows (follower_id, following_id) VALUES (?, ?)`
	if _, err := s.DB.Exec(query, followerID, followingID); err != nil {
		return fmt.Errorf("ChatService.Follow: %w", err)
	}
	log.Printf("User %d followed %d", followerID, followingID)
	return nil
}

func (s *ChatService) Unfollow(followerID, followingID int64) error {
	query := `DELETE FROM user_follows WHERE follower_id = ? AND following_id = ?`
	if _, err := s.DB.Exec(query, followerID, followingID); err != nil {
		return fmt.Errorf("ChatService.Unfollow: %w", err)
	}
	log.Printf("User %d unfollowed %d", followerID, followingID)
	return nil
}

func (s *ChatService) GetFollowers(userID int64) ([]model.User, error) {
	var users []model.User
	query := `SELECT u.* FROM users u
		JOIN user_follows uf ON u.id = uf.follower_id
		WHERE uf.following_id = ?
		ORDER BY uf.created_at DESC`
	if err := s.DB.Select(&users, query, userID); err != nil {
		return nil, fmt.Errorf("ChatService.GetFollowers: %w", err)
	}
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

func (s *ChatService) GetFollowing(userID int64) ([]model.User, error) {
	var users []model.User
	query := `SELECT u.* FROM users u
		JOIN user_follows uf ON u.id = uf.following_id
		WHERE uf.follower_id = ?
		ORDER BY uf.created_at DESC`
	if err := s.DB.Select(&users, query, userID); err != nil {
		return nil, fmt.Errorf("ChatService.GetFollowing: %w", err)
	}
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

func (s *ChatService) IsFollowing(followerID, followingID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_follows WHERE follower_id = ? AND following_id = ?`
	if err := s.DB.Get(&count, query, followerID, followingID); err != nil {
		return false, fmt.Errorf("ChatService.IsFollowing: %w", err)
	}
	return count > 0, nil
}

func (s *ChatService) GetFollowersCount(userID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_follows WHERE following_id = ?`
	if err := s.DB.Get(&count, query, userID); err != nil {
		return 0, fmt.Errorf("ChatService.GetFollowersCount: %w", err)
	}
	return count, nil
}

func (s *ChatService) GetFollowingCount(userID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_follows WHERE follower_id = ?`
	if err := s.DB.Get(&count, query, userID); err != nil {
		return 0, fmt.Errorf("ChatService.GetFollowingCount: %w", err)
	}
	return count, nil
}
