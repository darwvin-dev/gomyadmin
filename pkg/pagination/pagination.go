package pagination

import (
	"net/url"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 25
	MaxPerPage     = 100
)

type Page struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Offset  int `json:"offset"`
	Limit   int `json:"limit"`
}

// Parse converts query values into a bounded page description.
// per_page takes precedence over limit when both are present.
func Parse(values url.Values) Page {
	page := parsePositive(values.Get("page"), DefaultPage)
	perPage := parsePositive(first(values.Get("per_page"), values.Get("limit")), DefaultPerPage)
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Page{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
		Limit:   perPage,
	}
}

func Meta(page Page, total int64) map[string]any {
	pages := int64(0)
	if page.PerPage > 0 {
		pages = (total + int64(page.PerPage) - 1) / int64(page.PerPage)
	}
	return map[string]any{
		"page":     page.Page,
		"per_page": page.PerPage,
		"total":    total,
		"pages":    pages,
	}
}

func parsePositive(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
