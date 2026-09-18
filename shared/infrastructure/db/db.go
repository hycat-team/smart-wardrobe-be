package db

import (
	"context"
	"fmt"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/infrastructure/resilience"
	pkglogger "smart-wardrobe-be/shared/pkg/logger"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// supavisorPoolerPort is the fixed Supabase transaction-pooler port. Runtime
// traffic through it must avoid prepared statements and client startup
// parameters (FR-046); migration/LISTEN/advisory traffic uses direct 5432.
const supavisorPoolerPort = 6543

// queryExecMode resolves the pgx QueryExecMode for runtime traffic. The
// Supavisor transaction pooler (6543) cannot serve prepared statements, so it is
// forced to simple_protocol (FR-046). Direct connections use pgx's default
// (prepared statements, which also carry parameter OIDs) unless explicitly
// overridden via DATABASE_QUERY_EXEC_MODE.
func queryExecMode(cfg *config.Config) string {
	if cfg.Database.Port == supavisorPoolerPort {
		return "simple_protocol"
	}
	return strings.TrimSpace(cfg.Database.QueryExecMode)
}

func NewPostgresConnection(cfg *config.Config, l pkglogger.Interface) (*gorm.DB, error) {

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: true,
		},
	}

	var db *gorm.DB

	// search_path đa schema: mỗi module sở hữu schema riêng (Sprint 2).
	// `public` đặt ĐẦU để goose_db_version (bảng goose) luôn resolve ở public —
	// nếu không, goose CLI (`make migration-*` không kèm search_path) sẽ không
	// thấy bảng version đã được tạo qua embedded runner (search_path khác).
	const defaultSearchPath = "public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared"
	searchPath := cfg.Database.SearchPath
	if searchPath == "" {
		searchPath = defaultSearchPath
	}

	err := resilience.RunStartupRetry(cfg, l, "postgres", func(timeout time.Duration) error {
		// FR-046: Supavisor transaction pooler (6543) ignores/denies client
		// startup parameters, so search_path MUST come from the server side
		// (ALTER ROLE/DATABASE) there — never from this DSN. Direct
		// connections keep the DSN search_path for local/VPS convenience.
		// simple_protocol disables prepared statements, which the pooler cannot
		// serve; it is forced there by queryExecMode (FR-046).
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
		if cfg.Database.Port != supavisorPoolerPort {
			dsn += fmt.Sprintf(" search_path=%s", searchPath)
		}
		if mode := queryExecMode(cfg); mode != "" {
			dsn += fmt.Sprintf(" default_query_exec_mode=%s", mode)
		}

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

		sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
		sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
		db = openedDB
		return nil
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}
