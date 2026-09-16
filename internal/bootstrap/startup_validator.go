package bootstrap

import (
	sharedbootstrap "smart-wardrobe-be/shared/bootstrap"
)

// StartupValidator defines the contract for validations that must run before the application fully boots.
// 020: alias of the shared contract (single identity for wire + services).
type StartupValidator = sharedbootstrap.StartupValidator
