package models

type ProfilePriceMapping struct {
	ID           uint   `gorm:"primaryKey"                                json:"id"`
	RouterID     uint   `gorm:"not null;uniqueIndex:udx_router_profile"   json:"router_id"`
	ProfileName  string `gorm:"size:100;not null;uniqueIndex:udx_router_profile" json:"profile_name"`
	Price        int64  `gorm:"not null;default:0"                        json:"price"`
	SellingPrice int64  `gorm:"not null;default:0"                        json:"selling_price"`
	Validity     string `gorm:"size:20"                                   json:"validity"`
	ExpMode      string `gorm:"size:10"                                   json:"exp_mode"`
	LockUser     bool   `gorm:"not null;default:false"                    json:"lock_user"`
	LockServer   bool   `gorm:"not null;default:false"                    json:"lock_server"`

	Router Router `gorm:"foreignKey:RouterID" json:"-"`
}
