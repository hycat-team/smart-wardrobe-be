package dto

type UploadSignatureParams struct {
	Folder       string `json:"folder"`
	PublicID     string `json:"publicId"`
	ResourceType string `json:"resourceType"`
}

type UploadSignatureResult struct {
	Signature    string `json:"signature"`
	Timestamp    int64  `json:"timestamp"`
	ApiKey       string `json:"apiKey"`
	PublicID     string `json:"publicId"`
	Folder       string `json:"folder"`
	ResourceType string `json:"resourceType"`
}
