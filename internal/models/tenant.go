package models

import (
	"time"

	"gorm.io/gorm"
)

type TenantStatus string
type TenantPlan string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusTrial     TenantStatus = "trial"
	TenantStatusSuspended TenantStatus = "suspended"

	TenantPlanFree    TenantPlan = "free"
	TenantPlanStarter TenantPlan = "starter"
	TenantPlanPro     TenantPlan = "pro"
)

const PlatformTenantSlug = "__platform__"

type Tenant struct {
	ID        uint           `gorm:"primaryKey"                                  json:"id"`
	Name      string         `gorm:"size:100;not null;uniqueIndex"               json:"name"`
	Slug      string         `gorm:"size:100;not null;uniqueIndex"               json:"slug"`
	Plan      TenantPlan     `gorm:"type:varchar(20);not null;default:free"      json:"plan"`
	Status    TenantStatus   `gorm:"type:varchar(20);not null;default:active"    json:"status"`
	CreatedAt time.Time      `                                                   json:"created_at"`
	UpdatedAt time.Time      `                                                   json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                       json:"-"`

	Settings *TenantSettings `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"settings,omitempty"`
	Users    []User          `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
	Routers  []Router        `gorm:"foreignKey:TenantID;constraint:OnDelete:CASCADE" json:"-"`
}
