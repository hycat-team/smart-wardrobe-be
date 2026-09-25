package feedsort

type FeedSort string

const (
	Hot    FeedSort = "hot"
	Latest FeedSort = "latest"
)

func (s FeedSort) IsValid() bool {
	return s == Hot || s == Latest
}
