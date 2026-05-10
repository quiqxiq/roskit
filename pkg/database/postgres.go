package database

import (
	"fmt"

	"github.com/quiqxiq/roskit/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Settings{},
		&models.User{},
		&models.Router{},
		&models.ProfilePriceMapping{},
		&models.VoucherSale{},
		&models.PrintTemplate{},
		&models.AuditLog{},
	)
}
