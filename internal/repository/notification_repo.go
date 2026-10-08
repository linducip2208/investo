package repository

import (
	"fmt"
	"investo/internal/model"

	"github.com/jmoiron/sqlx"
)

type NotificationRepository struct {
	DB *sqlx.DB
}

func (r *NotificationRepository) Create(n *model.Notification) (int64, error) {
	query := `INSERT INTO notifications (user_id, type, title, message, link, is_read)
		VALUES (:user_id, :type, :title, :message, :link, :is_read)`
	result, err := r.DB.NamedExec(query, n)
	if err != nil {
		return 0, fmt.Errorf("NotificationRepository.Create: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("NotificationRepository.Create LastInsertId: %w", err)
	}
	return id, nil
}

func (r *NotificationRepository) FindByUserID(userID int64, limit int) ([]model.Notification, error) {
	var notifications []model.Notification
	query := `SELECT * FROM notifications WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`
	if err := r.DB.Select(&notifications, query, userID, limit); err != nil {
		return nil, fmt.Errorf("NotificationRepository.FindByUserID: %w", err)
	}
	return notifications, nil
}

func (r *NotificationRepository) MarkAsRead(userID, notifID int64) error {
	query := `UPDATE notifications SET is_read = true WHERE id = ? AND user_id = ?`
	_, err := r.DB.Exec(query, notifID, userID)
	if err != nil {
		return fmt.Errorf("NotificationRepository.MarkAsRead: %w", err)
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(userID int64) error {
	query := `UPDATE notifications SET is_read = true WHERE user_id = ? AND is_read = false`
	_, err := r.DB.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("NotificationRepository.MarkAllRead: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetUnreadCount(userID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = false`
	if err := r.DB.Get(&count, query, userID); err != nil {
		return 0, fmt.Errorf("NotificationRepository.GetUnreadCount: %w", err)
	}
	return count, nil
}
