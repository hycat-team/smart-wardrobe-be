package aienforcementmode

type AIEnforcementMode string

const (
	Strict      AIEnforcementMode = "strict"
	FreeOnly    AIEnforcementMode = "free_only"
	ObserveOnly AIEnforcementMode = "observe_only"
)
