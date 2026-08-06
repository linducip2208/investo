package model

import "time"

type ChatMessage struct {
	ID         int64     `json:"id" db:"id"`
	SenderID   int64     `json:"sender_id" db:"sender_id"`
	ReceiverID *int64    `json:"receiver_id,omitempty" db:"receiver_id"`
	Room       *string   `json:"room,omitempty" db:"room"`
	Message    string    `json:"message" db:"message"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	SenderName string    `json:"sender_name,omitempty" db:"sender_name"`
}

type UserFollow struct {
	FollowerID  int64     `json:"follower_id" db:"follower_id"`
	FollowingID int64     `json:"following_id" db:"following_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type EmailVerification struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	Type      string    `json:"type" db:"type"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Used      bool      `json:"used" db:"used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type DMConversation struct {
	UserID       int64     `json:"user_id"`
	UserName     string    `json:"user_name"`
	LastMessage  string    `json:"last_message"`
	LastMessageAt time.Time `json:"last_message_at"`
	UnreadCount  int       `json:"unread_count"`
}
