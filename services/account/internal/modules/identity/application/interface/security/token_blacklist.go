package security

import (
	sharedsecurity "smart-wardrobe-be/shared/application/interface/security"
)

// ITokenBlacklistService is the session-revocation port shared by every
// service (FR-023). It is aliased here so existing account references keep
// working; new code should import the shared package directly.
type ITokenBlacklistService = sharedsecurity.ITokenBlacklistService
