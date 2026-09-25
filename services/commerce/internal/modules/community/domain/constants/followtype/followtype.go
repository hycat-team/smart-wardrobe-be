package followtype

// FollowType filters a user's follow list. It is the API-facing label for the
// query param `type`; persisted columns keep the internal follower_id/followee_id
// naming. Per-item `relation` uses Following/Follower.
type FollowType string

const (
	Following FollowType = "following"
	Followers FollowType = "followers"
)

func (f FollowType) IsValid() bool {
	return f == Following || f == Followers
}

// Relation values returned per follow item.
const (
	RelationFollowing = "following"
	RelationFollower  = "follower"
)
