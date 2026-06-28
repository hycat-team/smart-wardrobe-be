package brandstatus

type BrandStatus string

const (
	PendingReview BrandStatus = "PENDING_REVIEW"
	Active        BrandStatus = "ACTIVE"
	Suspended     BrandStatus = "SUSPENDED"
	Archived      BrandStatus = "ARCHIVED"
)
