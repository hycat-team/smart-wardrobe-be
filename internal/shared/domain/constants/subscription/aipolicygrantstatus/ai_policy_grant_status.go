package aipolicygrantstatus

type AIPolicyGrantStatus string

const (
	Active AIPolicyGrantStatus = "active"
	Future AIPolicyGrantStatus = "future"
	Closed AIPolicyGrantStatus = "closed"
)
