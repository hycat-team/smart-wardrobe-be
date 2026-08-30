// Package messaging — outbox types are owned by the shared domain layer
// (feature 017); aliased here so existing infrastructure consumers keep
// compiling while implementation details stay in this package.
package messaging

import (
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

type OutboxStatus = shared_repos.OutboxStatus

const (
	OutboxStatusPending    = shared_repos.OutboxStatusPending
	OutboxStatusProcessing = shared_repos.OutboxStatusProcessing
	OutboxStatusPublished  = shared_repos.OutboxStatusPublished
	OutboxStatusFailed     = shared_repos.OutboxStatusFailed
)

type OutboxRecord = shared_repos.OutboxRecord

type OutboxRepository = shared_repos.OutboxRepository
