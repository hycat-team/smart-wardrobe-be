package itemtype

type ItemType int16

const (
	UserItem          ItemType = 0 // Personal item of the user
	SystemCatalogItem ItemType = 1 // System catalog item created by Admin
)
