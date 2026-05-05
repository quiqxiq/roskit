package models

import "time"

type VoucherSale struct {
	ID             uint      `gorm:"primaryKey"                                      json:"id"`
	TenantID       uint      `gorm:"not null;index;constraint:OnDelete:CASCADE"      json:"tenant_id"`
	RouterID       *uint     `gorm:"index;constraint:OnDelete:SET NULL"              json:"router_id"`
	SoldAt         time.Time `gorm:"not null;index"                                  json:"sold_at"`
	Username       string    `gorm:"size:100;not null;index"                         json:"username"`
	ProfileName    string    `gorm:"size:100;not null;index"                         json:"profile_name"`
	Price          int64     `gorm:"not null;default:0"                              json:"price"`
	SellingPrice   int64     `gorm:"not null;default:0"                              json:"selling_price"`
	Server         string    `gorm:"size:100;index"                                  json:"server"`
	IPAddress      string    `gorm:"size:45"                                         json:"ip_address"`
	MACAddress     string    `gorm:"size:17"                                         json:"mac_address"`
	Validity       string    `gorm:"size:20"                                         json:"validity"`
	IdempotencyKey string    `gorm:"type:char(64);not null;uniqueIndex"              json:"idempotency_key"`
	CreatedAt      time.Time `                                                       json:"created_at"`
}
