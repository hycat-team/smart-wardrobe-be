package mediatype

type MediaType string

const (
	Image MediaType = "image"
	Video MediaType = "video"
)

var (
	imageExtensions = []string{"jpg", "jpeg", "png", "webp", "avif", "gif"}
	videoExtensions = []string{"mp4", "webm", "mov", "m4v", "mkv", "avi"}
)

func (m MediaType) IsValid() bool {
	return m == Image || m == Video
}

// AllowedExtensions returns the file extensions accepted for this media type.
// The returned slice is a copy and is safe for the caller to mutate.
func (m MediaType) AllowedExtensions() []string {
	switch m {
	case Image:
		return append([]string(nil), imageExtensions...)
	case Video:
		return append([]string(nil), videoExtensions...)
	default:
		return nil
	}
}
