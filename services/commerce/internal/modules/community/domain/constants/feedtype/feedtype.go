package feedtype

type FeedType string

const (
	Explore   FeedType = "explore"
	Following FeedType = "following"
)

func (f FeedType) IsValid() bool {
	return f == Explore || f == Following
}
