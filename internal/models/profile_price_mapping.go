package models

type ProfilePriceMapping struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	ProfileName  string `gorm:"uniqueIndex;size:100;not null" json:"profile_name"`
	Price        int64  `gorm:"not null;default:0" json:"price"`
	SellingPrice int64  `gorm:"not null;default:0" json:"selling_price"`
	Validity     string `gorm:"size:20" json:"validity"`
	ExpMode      string `gorm:"size:10" json:"exp_mode"`
	LockUser     bool   `gorm:"default:false" json:"lock_user"`
	LockServer   bool   `gorm:"default:false" json:"lock_server"`
	RouterID     uint   `gorm:"index;not null" json:"router_id"`
	Router       Router `gorm:"foreignKey:RouterID" json:"-"`
}
