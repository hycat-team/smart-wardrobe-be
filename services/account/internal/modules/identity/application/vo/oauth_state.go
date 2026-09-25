package vo

// OAuthState is the server-side state persisted for the web redirect flow. It is
// stored under a one-time random key and consumed on callback.
type OAuthState struct {
	RedirectURL  string `json:"redirectUrl"`
	CodeVerifier string `json:"codeVerifier"`
}
