package models

import (
	"time"

	"gorm.io/gorm"
)

type SystemUser struct {
	ID           uint           `gorm:"primaryKey"                                                                        json:"id"`
	Username     string         `gorm:"size:100;not null;uniqueIndex:idx_system_users_username,where:deleted_at IS NULL"  json:"username"`
	PasswordHash string         `gorm:"size:255;not null"                                                                 json:"-"`
	Role         string         `gorm:"size:20;not null;default:admin"                                                    json:"role"`
	Active       bool           `gorm:"not null;default:true"                                                             json:"active"`
	LastLoginAt  *time.Time     `                                                                                         json:"last_login_at"`
	CreatedAt    time.Time      `                                                                                         json:"created_at"`
	UpdatedAt    time.Time      `                                                                                         json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"                                                                             json:"-"`
}
