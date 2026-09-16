package entities

func (User) TableName() string {
	return "users"
}

func (UserStyleProfile) TableName() string {
	return "user_style_profiles"
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
