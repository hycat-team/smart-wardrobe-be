package analyzetaskstatus

type AnalyzeTaskStatus string

const (
	Processing  AnalyzeTaskStatus = "processing"
	Completed   AnalyzeTaskStatus = "completed"
	Failed      AnalyzeTaskStatus = "failed"
	NeedsReview AnalyzeTaskStatus = "needs_review"
)
