package persistence

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aiusageeventstatus"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// The admin AI error count must cover every terminal error state, not just
// `released`: a provider call that exhausted its reservation without a verdict
// (`unknown_usage`) and a reservation that expired unverified
// (`expired_unverified`) are failed requests from the customer's point of
// view — the old query counted neither, so the error KPI read near zero.
func TestAdminAIUsageStatsCountsAllErrorStates(t *testing.T) {
	dbConn, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, dbConn.Exec(`DROP TABLE IF EXISTS ai_usage_events`).Error)
	require.NoError(t, dbConn.Exec(`
		CREATE TABLE ai_usage_events (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			user_id TEXT,
			operation TEXT NOT NULL DEFAULT '',
			logical_route TEXT NOT NULL DEFAULT '',
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			output_tokens INTEGER NOT NULL DEFAULT 0,
			thinking_tokens INTEGER NOT NULL DEFAULT 0,
			reserved_cost_micro_vnd INTEGER NOT NULL DEFAULT 0,
			actual_cost_micro_vnd INTEGER NOT NULL DEFAULT 0,
			estimated_max_cost_micro_vnd INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`).Error)
	t.Cleanup(func() { _ = dbConn.Exec(`DROP TABLE IF EXISTS ai_usage_events`).Error })

	vn := timeutils.VnLoc()
	inDay := func(h, m int) time.Time { return time.Date(2026, 10, 11, h, m, 0, 0, vn) }
	mkEvent := func(status aiusageeventstatus.AIUsageEventStatus, at time.Time, tokens int64) {
		require.NoError(t, dbConn.Exec(
			`INSERT INTO ai_usage_events (id, request_id, prompt_tokens, output_tokens, thinking_tokens, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), uuid.New().String(), tokens, 0, 0, string(status), at,
		).Error)
	}

	mkEvent(aiusageeventstatus.Confirmed, inDay(8, 0), 100)
	mkEvent(aiusageeventstatus.Reserved, inDay(8, 5), 0)
	mkEvent(aiusageeventstatus.InFlight, inDay(8, 10), 0)
	mkEvent(aiusageeventstatus.Released, inDay(9, 0), 10)
	mkEvent(aiusageeventstatus.UnknownUsage, inDay(9, 30), 20)
	mkEvent(aiusageeventstatus.ExpiredUnverified, inDay(10, 0), 30)
	// Previous VN day — must not leak into the count.
	mkEvent(aiusageeventstatus.Released, time.Date(2026, 10, 10, 20, 0, 0, 0, vn), 40)

	repo := NewAICostRepository(dbConn)
	stats, err := repo.GetAdminAIUsageStats(context.Background(), inDay(12, 0))
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 6, stats.AIRequestsCount, "only the six events of the VN day")
	assert.Equal(t, 3, stats.AIErrorCount, "released + unknown_usage + expired_unverified")
	assert.Equal(t, int64(160), stats.AIPaidTokens, "paid tokens sum the same six rows")
}
