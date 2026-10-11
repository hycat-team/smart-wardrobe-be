package entities

func (User) TableName() string {
	return "users"
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
