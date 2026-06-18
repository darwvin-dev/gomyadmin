// mongostore is an opt-in adapter module. It is intentionally separate from the
// core gomyadmin module so the core install path does not pull in mongo-driver.
module github.com/darwvin-dev/gomyadmin/pkg/adapters/mongostore

go 1.25.0

require (
	github.com/darwvin-dev/gomyadmin v1.0.3
	go.mongodb.org/mongo-driver v1.17.9
)

require (
	github.com/go-chi/chi/v5 v5.2.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

// Build against the in-repo core during development; external consumers ignore
// this replace and resolve the required version from the proxy.
replace github.com/darwvin-dev/gomyadmin => ../../..
