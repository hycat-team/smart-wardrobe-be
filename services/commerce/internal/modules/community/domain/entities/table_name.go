package entities

func (Post) TableName() string {
	return "posts"
}

func (PostMedia) TableName() string {
	return "post_media"
}

func (PostLike) TableName() string {
	return "post_likes"
}

func (PostComment) TableName() string {
	return "post_comments"
}

func (Follow) TableName() string {
	return "follows"
}

func (UserFollowStats) TableName() string {
	return "user_follow_stats"
}
