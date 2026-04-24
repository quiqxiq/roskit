package models

import "time"

type PrintTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Type      string    `gorm:"size:20;not null" json:"type"`
	Part      string    `gorm:"size:10;check:chk_part,part IN ('header','row','footer')" json:"part"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	RouterID  uint      `gorm:"index" json:"router_id"`
	Router    Router    `gorm:"foreignKey:RouterID" json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}
