package models

import "time"

type VoucherSale struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SoldAt         time.Time `gorm:"index;not null" json:"sold_at"`
	Username       string    `gorm:"size:100;not null;index" json:"username"`
	ProfileName    string    `gorm:"size:100;not null" json:"profile_name"`
	Price          int64     `gorm:"not null;default:0" json:"price"`
	SellingPrice   int64     `gorm:"not null;default:0" json:"selling_price"`
	Server         string    `gorm:"size:100;index" json:"server"`
	IPAddress      string    `gorm:"size:45" json:"ip_address"`
	MACAddress     string    `gorm:"size:17" json:"mac_address"`
	Validity       string    `gorm:"size:20" json:"validity"`
	IdempotencyKey string    `gorm:"size:200;uniqueIndex" json:"idempotency_key"`
	RouterID       uint      `gorm:"index;not null" json:"router_id"`
	Router         Router    `gorm:"foreignKey:RouterID" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}
