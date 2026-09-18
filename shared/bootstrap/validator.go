// Package bootstrap holds the shared boot-time contracts used by every
// service binary and, transitionally, the monolith (020).
package bootstrap

import "context"

// StartupValidator defines the contract for validations that must run before
// the application fully boots. It lives in shared (not in any service) so
// application layers can depend on it without importing a composition root.
type StartupValidator interface {
	Validate(ctx context.Context) error
}
