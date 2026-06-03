package itemtype

type ItemType int16

const (
	UserItem          ItemType = 0 // Trang phục cá nhân của user
	SystemCatalogItem ItemType = 1 // Trang phục mẫu của hệ thống do Admin tạo
)
