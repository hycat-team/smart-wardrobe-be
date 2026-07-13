package brandstatus

type BrandStatus string

const (
	PendingReview BrandStatus = "pending_review"
	Active        BrandStatus = "active"
	Suspended     BrandStatus = "suspended"
	Archived      BrandStatus = "archived"
)
