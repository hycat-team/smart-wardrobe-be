// Package analysisreason defines the canonical reason codes attached to fashion
// item analysis. Reasons are stable codes (not human messages) so the frontend
// can map them to localized text (feature 023, FR-030).
package analysisreason

// Review reasons accompany the needs_review state (fixable by the user).
const (
	// ReviewUncertainCategory: the image has a single item but the AI is not
	// confident about its category. The user may pick a category and confirm.
	ReviewUncertainCategory = "uncertain_category"
)

// Processing-error reasons accompany the failed state.
const (
	// ErrorMultipleItems: the image contains multiple items.
	ErrorMultipleItems = "multiple_items_detected"
	// ErrorFullBodyOutfit: the image is a full-body/outfit shot.
	ErrorFullBodyOutfit = "full_body_outfit_detected"
	// ErrorAnalysisTemporary: a transient analysis/system error.
	ErrorAnalysisTemporary = "analysis_temporary_error"
	// ErrorAutoRetryExceeded: automatic retries were exhausted.
	ErrorAutoRetryExceeded = "auto_retry_exceeded"
)

// InvalidImageReasons lists the reasons that mark an unusable image; retries are
// blocked for these (FR-033) because re-running on the same image is futile.
var InvalidImageReasons = map[string]bool{
	ErrorMultipleItems:  true,
	ErrorFullBodyOutfit: true,
}

// IsInvalidImageReason reports whether a reason code denotes an unusable image.
func IsInvalidImageReason(reason string) bool {
	return InvalidImageReasons[reason]
}
