package plankind

type PlanKind int16

const (
	DefaultFree PlanKind = 0
	Finite      PlanKind = 1
	Lifetime    PlanKind = 2
)
