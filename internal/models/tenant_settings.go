package models

import "time"

type TenantSettings struct {
	ID           uint      `gorm:"primaryKey"                                         json:"id"`
	TenantID     uint      `gorm:"not null;uniqueIndex;constraint:OnDelete:CASCADE"    json:"tenant_id"`
	HotspotName  string    `gorm:"size:100"                                           json:"hotspot_name"`
	DNSName      string    `gorm:"size:100"                                           json:"dns_name"`
	Currency     string    `gorm:"size:10;not null;default:Rp"                        json:"currency"`
	Phone        string    `gorm:"size:20"                                            json:"phone"`
	Email        string    `gorm:"size:100"                                           json:"email"`
	InfoLP       string    `gorm:"type:text"                                          json:"info_lp"`
	IdleTimeout  int       `gorm:"not null;default:30"                                json:"idle_timeout"`
	ReportMode   string    `gorm:"size:20;not null;default:disable"                   json:"report_mode"`
	WebhookToken string    `gorm:"size:255"                                           json:"-"`
	LogoPath     string    `gorm:"type:text;not null;default:''"                      json:"logo_path"`
	Timezone     string    `gorm:"size:50;not null;default:''"                        json:"timezone"`
	CreatedAt    time.Time `                                                          json:"created_at"`
	UpdatedAt    time.Time `                                                          json:"updated_at"`
}
