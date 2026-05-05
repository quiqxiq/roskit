package models

import (
	"time"

	"gorm.io/gorm"
)

type Router struct {
	ID                   uint           `gorm:"primaryKey"                                                              json:"id"`
	Name                 string         `gorm:"size:100;not null;uniqueIndex:idx_routers_name,where:deleted_at IS NULL" json:"name"`
	IPAddress            string         `gorm:"size:45;not null"                                                        json:"ip_address"`
	APIPort              int            `gorm:"not null;default:8728"                                                   json:"api_port"`
	APIUsername          string         `gorm:"size:100;not null"                                                       json:"api_username"`
	APIPasswordEncrypted string         `gorm:"column:password;not null"                                                json:"-"`
	SSHPort              *int           `gorm:"default:22"                                                              json:"ssh_port"`
	SSHUsername          *string        `gorm:"size:50"                                                                 json:"ssh_username"`
	SSHPasswordEncrypted *string        `gorm:"column:ssh_password"                                                     json:"-"`
	Status               RouterStatus   `gorm:"type:varchar(20);not null;default:unknown"                               json:"status"`
	LastSeenAt           *time.Time     `                                                                               json:"last_seen_at"`
	Notes                *string        `gorm:"type:text"                                                               json:"notes"`
	CreatedAt            time.Time      `                                                                               json:"created_at"`
	UpdatedAt            time.Time      `                                                                               json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index"                                                                   json:"-"`

	HotspotConfig *HotspotConfig `gorm:"foreignKey:RouterID" json:"hotspot_config,omitempty"`
}
