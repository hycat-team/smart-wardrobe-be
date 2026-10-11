package useractivitytype

type UserActivityType string

const (
	// Login is recorded when a session is issued through password or Google
	// login.
	Login UserActivityType = "login"
	// Refresh is recorded on refresh-token rotation, which a client performs
	// about once per access-token expiry while the app is actually in use.
	Refresh UserActivityType = "refresh"
)
