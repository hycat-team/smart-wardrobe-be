package postitemstatus

type PostItemStatus int16

const (
	Hidden    PostItemStatus = 0
	Available PostItemStatus = 1
	Sold      PostItemStatus = 2
)
