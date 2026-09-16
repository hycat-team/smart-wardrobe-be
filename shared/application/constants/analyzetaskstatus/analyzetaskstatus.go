package analyzetaskstatus

type AnalyzeTaskStatus string

const (
	Completed   AnalyzeTaskStatus = "completed"
	Failed      AnalyzeTaskStatus = "failed"
	NeedsReview AnalyzeTaskStatus = "needs_review"
)
