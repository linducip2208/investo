package service

import (
	"investo/internal/model"
	"investo/internal/repository"
	"sync"
	"time"
)

type NotificationService struct {
	Repo  *repository.NotificationRepository
	Cache []model.Notification
	mu    sync.RWMutex
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		Repo:  repo,
		Cache: make([]model.Notification, 0),
	}
}

func (s *NotificationService) Add(userID int64, notifType, title, message, link string) error {
	n := &model.Notification{
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		Link:      link,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	id, err := s.Repo.Create(n)
	if err != nil {
		return err
	}

	n.ID = id

	s.mu.Lock()
	s.Cache = append([]model.Notification{*n}, s.Cache...)
	if len(s.Cache) > 100 {
		s.Cache = s.Cache[:100]
	}
	s.mu.Unlock()

	return nil
}

func (s *NotificationService) GetUserNotifications(userID int64, limit int) []model.Notification {
	if limit <= 0 {
		limit = 10
	}

	notifications, err := s.Repo.FindByUserID(userID, limit)
	if err != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()

		var result []model.Notification
		for _, n := range s.Cache {
			if n.UserID == userID {
				result = append(result, n)
			}
		}
		if len(result) > limit {
			result = result[:limit]
		}
		return result
	}

	return notifications
}

func (s *NotificationService) MarkAsRead(userID, notifID int64) error {
	if err := s.Repo.MarkAsRead(userID, notifID); err != nil {
		return err
	}

	s.mu.Lock()
	for i, n := range s.Cache {
		if n.ID == notifID && n.UserID == userID {
			s.Cache[i].IsRead = true
			break
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *NotificationService) MarkAllRead(userID int64) error {
	if err := s.Repo.MarkAllRead(userID); err != nil {
		return err
	}

	s.mu.Lock()
	for i, n := range s.Cache {
		if n.UserID == userID {
			s.Cache[i].IsRead = true
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *NotificationService) GetUnreadCount(userID int64) int {
	count, err := s.Repo.GetUnreadCount(userID)
	if err != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()

		count = 0
		for _, n := range s.Cache {
			if n.UserID == userID && !n.IsRead {
				count++
			}
		}
		return count
	}
	return count
}
