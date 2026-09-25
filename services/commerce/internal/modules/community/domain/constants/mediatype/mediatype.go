package mediatype

type MediaType string

const (
	Image MediaType = "image"
	Video MediaType = "video"
)

func (m MediaType) IsValid() bool {
	return m == Image || m == Video
}
