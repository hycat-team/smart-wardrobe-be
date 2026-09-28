package dto

type UploadSignatureParams struct {
	Folder         string `json:"folder"`
	PublicID       string `json:"publicId"`
	ResourceType   string `json:"resourceType"`
	AllowedFormats string `json:"allowedFormats"`
}

type UploadSignatureResult struct {
	Signature    string `json:"signature"`
	Timestamp    int64  `json:"timestamp"`
	ApiKey       string `json:"apiKey"`
	PublicID     string `json:"publicId,omitempty"`
	Folder       string `json:"folder"`
	ResourceType string `json:"resourceType"`
	// AllowedFormats must be sent back verbatim in the upload form; Cloudinary
	// recomputes the signature from the received fields, so omitting it fails.
	AllowedFormats string `json:"allowedFormats,omitempty"`
}
