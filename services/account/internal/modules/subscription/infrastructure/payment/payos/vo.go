package payos

type payOSResponseData struct {
	CheckoutUrl string `json:"checkoutUrl"`
}

type payOSResponse struct {
	Code string            `json:"code"`
	Desc string            `json:"desc"`
	Data payOSResponseData `json:"data"`
}
