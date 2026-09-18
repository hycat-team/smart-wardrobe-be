package votetype

type VoteType string

const (
	Like          VoteType = "like"
	Dislike       VoteType = "dislike"
	WouldBuy      VoteType = "would_buy"
	NotInterested VoteType = "not_interested"
)
