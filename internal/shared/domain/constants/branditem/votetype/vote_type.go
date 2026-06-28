package votetype

type VoteType string

const (
	Like          VoteType = "LIKE"
	Dislike       VoteType = "DISLIKE"
	WouldBuy      VoteType = "WOULD_BUY"
	NotInterested VoteType = "NOT_INTERESTED"
)
