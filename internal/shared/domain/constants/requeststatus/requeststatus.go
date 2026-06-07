package requeststatus

type RequestStatus int16

const (
	Pending  RequestStatus = 0
	Accepted RequestStatus = 1
	Rejected RequestStatus = 2
	Canceled RequestStatus = 3
)
