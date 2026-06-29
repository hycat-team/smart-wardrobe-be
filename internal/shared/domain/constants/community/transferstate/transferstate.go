package transferstate

type TransferState int16

const (
	None     TransferState = 0 // Not transferred yet
	Pending  TransferState = 1 // Pending receipt of item
	Accepted TransferState = 2 // Accepted receipt of item
	Declined TransferState = 3 // Declined receipt of item
)
