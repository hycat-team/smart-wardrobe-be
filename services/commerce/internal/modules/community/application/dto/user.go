package dto

import "github.com/google/uuid"

// CommunityUserRes is the canonical user projection used across community
// responses. It is embedded as a nested `user` object when user info appears
// alongside other data (post, comment, follow), and used directly for
// user-only lists (likes, search results, public profile).
type CommunityUserRes struct {
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	AvatarURL *string   `json:"avatarUrl,omitempty"`
	Gender    *int      `json:"gender,omitempty"`
}
