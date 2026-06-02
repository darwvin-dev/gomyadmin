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

func TestParseDefaults(t *testing.T) {
	page := Parse(url.Values{})
	if page.Page != DefaultPage {
		t.Fatalf("page = %d, want %d", page.Page, DefaultPage)
	}
	if page.PerPage != DefaultPerPage {
		t.Fatalf("per_page = %d, want %d", page.PerPage, DefaultPerPage)
	}
	if page.Offset != 0 {
		t.Fatalf("offset = %d, want 0", page.Offset)
	}
	if page.Limit != DefaultPerPage {
		t.Fatalf("limit = %d, want %d", page.Limit, DefaultPerPage)
	}
}

func TestParseZeroPageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"page": {"0"}})
	if page.Page != DefaultPage {
		t.Fatalf("page = %d, want %d", page.Page, DefaultPage)
	}
}

func TestParseNegativePageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"page": {"-5"}})
	if page.Page != DefaultPage {
		t.Fatalf("page = %d, want %d", page.Page, DefaultPage)
	}
}

func TestParseZeroPerPageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"per_page": {"0"}})
	if page.PerPage != DefaultPerPage {
		t.Fatalf("per_page = %d, want %d", page.PerPage, DefaultPerPage)
	}
}

func TestParseNegativePerPageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"per_page": {"-10"}})
	if page.PerPage != DefaultPerPage {
		t.Fatalf("per_page = %d, want %d", page.PerPage, DefaultPerPage)
	}
}

func TestParseNonNumericPageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"page": {"abc"}})
	if page.Page != DefaultPage {
		t.Fatalf("page = %d, want %d", page.Page, DefaultPage)
	}
}

func TestParseNonNumericPerPageFallsBackToDefault(t *testing.T) {
	page := Parse(url.Values{"per_page": {"notanumber"}})
	if page.PerPage != DefaultPerPage {
		t.Fatalf("per_page = %d, want %d", page.PerPage, DefaultPerPage)
	}
}

func TestParseLimitFallbackWhenPerPageAbsent(t *testing.T) {
	page := Parse(url.Values{"limit": {"10"}})
	if page.PerPage != 10 {
		t.Fatalf("per_page = %d, want 10", page.PerPage)
	}
	if page.Limit != 10 {
		t.Fatalf("limit = %d, want 10", page.Limit)
	}
}

func TestParsePerPageTakesPrecedenceOverLimit(t *testing.T) {
	page := Parse(url.Values{"per_page": {"20"}, "limit": {"50"}})
	if page.PerPage != 20 {
		t.Fatalf("per_page = %d, want 20 (per_page should take precedence over limit)", page.PerPage)
	}
}

func TestParseOffsetCalculation(t *testing.T) {
	page := Parse(url.Values{"page": {"4"}, "per_page": {"10"}})
	if page.Offset != 30 {
		t.Fatalf("offset = %d, want 30", page.Offset)
	}
}

func TestMetaTotalPages(t *testing.T) {
	p := Page{Page: 1, PerPage: 10, Offset: 0, Limit: 10}
	meta := Meta(p, 25)
	if meta["pages"] != int64(3) {
		t.Fatalf("pages = %v, want 3", meta["pages"])
	}
	if meta["total"] != int64(25) {
		t.Fatalf("total = %v, want 25", meta["total"])
	}
}

func TestMetaExactDivision(t *testing.T) {
	p := Page{Page: 1, PerPage: 10, Offset: 0, Limit: 10}
	meta := Meta(p, 20)
	if meta["pages"] != int64(2) {
		t.Fatalf("pages = %v, want 2", meta["pages"])
	}
}

func TestMetaZeroTotal(t *testing.T) {
	p := Page{Page: 1, PerPage: 25, Offset: 0, Limit: 25}
	meta := Meta(p, 0)
	if meta["pages"] != int64(0) {
		t.Fatalf("pages = %v, want 0", meta["pages"])
	}
}
