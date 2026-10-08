package model

import "time"

type AuditLog struct {
	ID          int64     `db:"id"`
	ActorUserID *int64    `db:"actor_user_id"`
	Action      string    `db:"action"`
	Entity      string    `db:"entity"`
	EntityID    string    `db:"entity_id"`
	Meta        string    `db:"meta"`
	IP          string    `db:"ip"`
	CreatedAt   time.Time `db:"created_at"`
}
