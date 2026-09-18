package dto

type UploadSignatureParams struct {
	Folder   string `json:"folder"`
	PublicID string `json:"publicId"`
}

type UploadSignatureResult struct {
	Signature string `json:"signature"`
	Timestamp int64  `json:"timestamp"`
	ApiKey    string `json:"apiKey"`
	PublicID  string `json:"publicId"`
	Folder    string `json:"folder"`
}
