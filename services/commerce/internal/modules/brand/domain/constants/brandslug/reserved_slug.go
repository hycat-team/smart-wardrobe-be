package brandslug

import "strings"

var reserved = map[string]struct{}{
	"admin":          {},
	"ai-stylist":     {},
	"api":            {},
	"auth":           {},
	"brand":          {},
	"brand-portal":   {},
	"brands":         {},
	"cart":           {},
	"checkout":       {},
	"community":      {},
	"delete-account": {},
	"loyalties":      {},
	"marketplace":    {},
	"payment":        {},
	"posts":          {},
	"pricing":        {},
	"privacy":        {},
	"products":       {},
	"profile":        {},
	"returns":        {},
	"support":        {},
	"users":          {},
	"wardrobe":       {},
	"workpage":       {},
}

func IsReserved(slug string) bool {
	_, found := reserved[strings.ToLower(strings.TrimSpace(slug))]
	return found
}
