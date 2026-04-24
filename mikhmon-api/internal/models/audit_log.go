package models

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Action    string    `gorm:"size:50;not null;index" json:"action"`
	Entity    string    `gorm:"size:50;not null" json:"entity"`
	EntityID  string    `gorm:"size:50" json:"entity_id"`
	Details   string    `gorm:"type:text" json:"details"`
	IPAddress string    `gorm:"size:45" json:"ip_address"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}
