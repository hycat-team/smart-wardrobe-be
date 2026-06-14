package db

import (
	"context"
	"fmt"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/infrastructure/resilience"
	pkglogger "smart-wardrobe-be/pkg/logger"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func NewPostgresConnection(cfg *config.Config, l pkglogger.Interface) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
		},
	}

	var db *gorm.DB

	err := resilience.RunStartupRetry(cfg, l, "postgres", func(timeout time.Duration) error {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s connect_timeout=%d",
			cfg.Database.Host,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DbName,
			cfg.Database.Port,
			cfg.Database.SslMode,
			cfg.Database.TimeZone,
			int(timeout.Seconds()),
		)

		openedDB, err := gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to open database connection: %w", err)
		}

		sqlDB, err := openedDB.DB()
		if err != nil {
			return fmt.Errorf("failed to get database instance: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := sqlDB.PingContext(ctx); err != nil {
			_ = sqlDB.Close()
			return fmt.Errorf("failed to ping database: %w", err)
		}

		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
		db = openedDB
		return nil
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}
