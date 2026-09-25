package posttype

type PostType string

const (
	Outfit PostType = "outfit"
	Media  PostType = "media"
)

func (p PostType) IsValid() bool {
	return p == Outfit || p == Media
}
