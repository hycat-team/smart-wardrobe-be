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
