package models

import (
	"time"

	"gorm.io/gorm"
)

type Router struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	SessionName string         `gorm:"size:100;not null" json:"session_name"`
	IP          string         `gorm:"size:45;not null" json:"ip"`
	Username    string         `gorm:"size:100;not null" json:"username"`
	PasswordEnc string         `gorm:"column:password;size:500;not null" json:"-"`
	HotspotName string         `gorm:"size:100" json:"hotspot_name"`
	DNSName     string         `gorm:"size:100" json:"dns_name"`
	Currency    string         `gorm:"size:10;default:Rp" json:"currency"`
	Phone       string         `gorm:"size:20" json:"phone"`
	Email       string         `gorm:"size:100" json:"email"`
	InfoLP      string         `gorm:"size:255" json:"info_lp"`
	IdleTimeout string         `gorm:"size:10;default:30" json:"idle_timeout"`
	ReportMode  string         `gorm:"size:20;default:disable" json:"report_mode"`
	Token       string         `gorm:"size:255" json:"-"`
	Timezone    string         `gorm:"size:50;default:''" json:"timezone"`
	LogoPath    string         `gorm:"size:500;default:''" json:"logo_path"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
