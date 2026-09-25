package poststatus

type PostStatus string

const (
	Published PostStatus = "published"
	Hidden    PostStatus = "hidden"
	Deleted   PostStatus = "deleted"
)

func (s PostStatus) IsValid() bool {
	return s == Published || s == Hidden || s == Deleted
}
