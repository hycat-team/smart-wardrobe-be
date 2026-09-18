package db

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// stubLogger satisfies logger.Interface without emitting anything.
type stubLogger struct{}

func (stubLogger) Debug(string, ...zap.Field) {}
func (stubLogger) Info(string, ...zap.Field)  {}
func (stubLogger) Warn(string, ...zap.Field)  {}
func (stubLogger) Error(string, ...zap.Field) {}
func (stubLogger) Fatal(string, ...zap.Field) {}

// TestNewOutboxListenerDisabledNoOp proves FR-009: with the listener off, boot
// gets a quiet no-op listener instead of an error.
func TestNewOutboxListenerDisabledNoOp(t *testing.T) {
	cfg := &config.Config{}
	cfg.Outbox.ListenerEnabled = false

	ol, err := NewOutboxListener(cfg, stubLogger{})
	require.NoError(t, err)
	require.NotNil(t, ol)
	defer ol.Stop()

	select {
	case <-ol.SignalChan():
		t.Fatal("disabled listener must never signal")
	case <-time.After(100 * time.Millisecond):
	}
}

// TestNewOutboxListenerUnreachableNonFatal proves FR-009/FR-010: a dead
// LISTEN channel (pooler without support, refused connection) degrades to
// poll mode instead of failing boot.
func TestNewOutboxListenerUnreachableNonFatal(t *testing.T) {
	cfg := &config.Config{}
	cfg.Outbox.ListenerEnabled = true
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = 15999 // closed: connection refused immediately
	cfg.Database.User = "nobody"
	cfg.Database.DbName = "nowhere"

	ol, err := NewOutboxListener(cfg, stubLogger{})
	require.NoError(t, err, "unreachable LISTEN must not fail boot")
	require.NotNil(t, ol)
	defer ol.Stop()

	select {
	case <-ol.SignalChan():
		t.Fatal("failed listener must never signal")
	case <-time.After(100 * time.Millisecond):
	}
}

// TestRunMigrationsIfEnabledSkipped proves FR-011: disabled profiles boot
// without touching the database.
func TestRunMigrationsIfEnabledSkipped(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.MigrateOnStart = false

	assert.NoError(t, RunMigrationsIfEnabled(context.Background(), cfg, nil, stubLogger{}))
}

// TestRunMigrationsIfEnabledNoTreeSkips proves a service without an embedded
// migration tree is skipped (not an error).
func TestRunMigrationsIfEnabledNoTreeSkips(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.MigrateOnStart = true

	assert.NoError(t, RunMigrationsIfEnabled(context.Background(), cfg, nil, stubLogger{}))
}

// TestRunMigrationsIfEnabledSurfacesDirectOpenError proves the gate only skips:
// an enabled profile whose DIRECT migration connection cannot be opened must
// surface the error, not swallow it.
func TestRunMigrationsIfEnabledSurfacesDirectOpenError(t *testing.T) {
	orig := openMigrationDB
	t.Cleanup(func() { openMigrationDB = orig })
	openMigrationDB = func(*config.Config) (*gorm.DB, error) {
		return nil, errors.New("dial direct failed")
	}

	cfg := &config.Config{}
	cfg.Database.MigrateOnStart = true

	assert.Error(t, RunMigrationsIfEnabled(context.Background(), cfg, fstest.MapFS{}, stubLogger{}))
}
