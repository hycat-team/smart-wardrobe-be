package bootstrap

import "context"

// StartupValidator defines the contract for validations that must run before the application fully boots.
type StartupValidator interface {
	Validate(ctx context.Context) error
}
