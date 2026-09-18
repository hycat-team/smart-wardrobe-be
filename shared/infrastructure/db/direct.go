package db

import (
	"fmt"

	"smart-wardrobe-be/shared/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DirectDSN builds the session-scoped DSN (direct connection, FR-046) from the
// listener fields, falling back to the runtime database fields when unset.
//
// Migration, LISTEN/NOTIFY and advisory locks are session-scoped and MUST use
// this DSN — never the runtime transaction pooler (`:6543`), which does not
// preserve session state.
func DirectDSN(cfg *config.Config) string {
	host := cfg.Database.ListenerHost
	if host == "" {
		host = cfg.Database.Host
	}
	port := cfg.Database.ListenerPort
	if port == 0 {
		port = cfg.Database.Port
	}
	sslMode := cfg.Database.ListenerSslMode
	if sslMode == "" {
		sslMode = cfg.Database.SslMode
	}
	timeZone := cfg.Database.TimeZone
	if timeZone == "" {
		timeZone = "Asia/Ho_Chi_Minh"
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DbName,
		port,
		sslMode,
		timeZone,
	)
}

// OpenDirectDB opens a small pool on the direct DSN, used for session-scoped
// work (migration, advisory locks). Callers MUST close the underlying sql.DB.
func OpenDirectDB(cfg *config.Config) (*gorm.DB, error) {
	opened, err := gorm.Open(postgres.Open(DirectDSN(cfg)), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open direct connection: %w", err)
	}
	sqlDB, err := opened.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve direct connection: %w", err)
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(2)
	return opened, nil
}
