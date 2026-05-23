package pagination

import (
	"net/url"
	"testing"
)

func TestParseCapsPerPage(t *testing.T) {
	values := url.Values{"page": {"3"}, "per_page": {"500"}}
	page := Parse(values)
	if page.Page != 3 {
		t.Fatalf("page = %d", page.Page)
	}
	if page.PerPage != MaxPerPage {
		t.Fatalf("per page = %d", page.PerPage)
	}
	if page.Offset != 200 {
		t.Fatalf("offset = %d", page.Offset)
	}
}
