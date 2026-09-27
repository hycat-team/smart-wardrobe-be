package branditemstatus

type BrandItemStatus string

const (
	Draft    BrandItemStatus = "draft"
	Active   BrandItemStatus = "active"
	Archived BrandItemStatus = "archived"
	// Failed marks a system analysis error. It is terminal for manual status
	// changes; only retry-analysis (or the system) may move an item out of it.
	Failed BrandItemStatus = "failed"
)
