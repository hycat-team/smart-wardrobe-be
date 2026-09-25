package authprovider

// AuthProvider identifies the identity provider a user account was created
// with. Only two values are supported: local (email/password) and Google.
type AuthProvider string

const (
	Local  AuthProvider = "local"
	Google AuthProvider = "google"
)
