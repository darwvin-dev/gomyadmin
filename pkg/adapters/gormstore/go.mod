// gormstore is an opt-in adapter module. It is intentionally separate from the
// core gomyadmin module so the core install path does not pull in GORM.
module github.com/darwvin-dev/gomyadmin/pkg/adapters/gormstore

go 1.25.0

require (
	github.com/darwvin-dev/gomyadmin v1.0.3
	github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore v1.0.3
	github.com/glebarez/sqlite v1.11.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-chi/chi/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	modernc.org/libc v1.72.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.49.1 // indirect
)

// Build against the in-repo core and sibling adapter during development;
// external consumers ignore these replaces and resolve from the proxy.
replace github.com/darwvin-dev/gomyadmin => ../../..

replace github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore => ../sqlstore
