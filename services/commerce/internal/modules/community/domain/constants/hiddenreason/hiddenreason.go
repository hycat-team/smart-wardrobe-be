package hiddenreason

type HiddenReason string

const (
	Admin          HiddenReason = "admin"
	AuthorDisabled HiddenReason = "author_disabled"
)

func (h HiddenReason) IsValid() bool {
	return h == Admin || h == AuthorDisabled
}
