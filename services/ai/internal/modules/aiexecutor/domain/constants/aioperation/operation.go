package aioperation

type AIOperation string

const (
	Chat      AIOperation = "chat"
	Outfit    AIOperation = "outfit"
	Summary   AIOperation = "summary"
	Rewriter  AIOperation = "rewriter"
	Vision    AIOperation = "vision"
	Embedding AIOperation = "embedding"
)

// alwaysFreeOperations are the operations that never reach a paid model, so they
// bypass the cost policy entirely and are always tracked on the free route.
//
// The set is declared once, here, because the orchestrator and the cost use case
// both need it and two copies of the same list is how they drift apart, with one
// of them quietly starting to meter an operation that was never metered.
var alwaysFreeOperations = map[AIOperation]bool{
	Vision:    true,
	Embedding: true,
}

// IsAlwaysFree reports whether op bypasses the cost policy because it cannot
// consume a paid model.
func IsAlwaysFree(op AIOperation) bool {
	return alwaysFreeOperations[op]
}
