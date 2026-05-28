package depositstatus

type DepositStatus int16

const (
	Pending DepositStatus = 0
	Success DepositStatus = 1
	Failed  DepositStatus = 2
)
