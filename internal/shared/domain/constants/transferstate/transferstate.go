package transferstate

type TransferState int16

const (
	None     TransferState = 0 // Chưa bàn giao
	Pending  TransferState = 1 // Đang chờ nhận đồ
	Accepted TransferState = 2 // Đã chấp nhận nhận đồ
	Declined TransferState = 3 // Đã từ chối nhận đồ
)
