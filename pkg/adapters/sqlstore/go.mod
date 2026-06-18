// sqlstore is an opt-in adapter module. It is intentionally separate from the
// core gomyadmin module so the core install path stays lightweight.
module github.com/darwvin-dev/gomyadmin/pkg/adapters/sqlstore

go 1.25.0

require (
	github.com/darwvin-dev/gomyadmin v1.0.3
	modernc.org/sqlite v1.49.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-chi/chi/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
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
)

// Build against the in-repo core during development; external consumers ignore
// this replace and resolve the required version from the proxy.
replace github.com/darwvin-dev/gomyadmin => ../../..
