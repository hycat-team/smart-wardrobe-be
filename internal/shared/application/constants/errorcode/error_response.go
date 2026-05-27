package errorcode

type ErrorResponse struct {
	Status int    `json:"status"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return e.Detail
}

func NewErrorResponse(status int, title, detail string) error {
	return &ErrorResponse{
		Status: status,
		Title:  title,
		Detail: detail,
	}
}
