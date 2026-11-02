package storage

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"anntodo/pkg/models"
	"anntodo/pkg/settings"
)

// Open connects to the configured database and migrates the schema.
func Open(cfg *settings.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Database.Type {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		// WAL mode + busy timeout give good behaviour for a small multi-user server.
		dsn := cfg.Database.Path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)"
		dialector = sqlite.Open(dsn)
	case "mysql":
		port := cfg.Database.Port
		if port == 0 {
			port = 3306
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.Database.User, cfg.Database.Password, cfg.Database.Host, port, cfg.Database.Name)
		dialector = mysql.Open(dsn)
	case "postgres":
		port := cfg.Database.Port
		if port == 0 {
			port = 5432
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			cfg.Database.Host, port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type %q", cfg.Database.Type)
	}

	lvl := logger.Warn
	if cfg.Log.Level == "debug" {
		lvl = logger.Info
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		NowFunc: func() time.Time { return time.Now().UTC() },
		Logger:  logger.New(slogWriter{}, logger.Config{SlowThreshold: 500 * time.Millisecond, LogLevel: lvl, IgnoreRecordNotFoundError: true}),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if cfg.Database.Type == "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		// SQLite allows a single writer; one connection avoids "database is locked".
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(models.All()...); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}

type slogWriter struct{}

func (slogWriter) Printf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...), "component", "gorm")
}
