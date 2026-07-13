package depositstatus

type DepositStatus int16

const (
	Pending                DepositStatus = 0
	Success                DepositStatus = 1
	FailedLegacy           DepositStatus = 2
	Creating               DepositStatus = 3
	ReconciliationRequired DepositStatus = 4
	Reconciling            DepositStatus = 5
	CreationFailed         DepositStatus = 6
	Cancelled              DepositStatus = 7
	Expired                DepositStatus = 8
	InvestigationRequired  DepositStatus = 9
)

var ActivePaymentStatuses = []DepositStatus{Creating, Pending, ReconciliationRequired, Reconciling}
