// redisstore is an opt-in adapter module. It is intentionally separate from the
// core gomyadmin module so the core install path does not pull in go-redis.
module github.com/darwvin-dev/gomyadmin/pkg/adapters/redisstore

go 1.25.0

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/darwvin-dev/gomyadmin v1.0.3
	github.com/redis/go-redis/v9 v9.18.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

// Build against the in-repo core during development; external consumers ignore
// this replace and resolve the required version from the proxy.
replace github.com/darwvin-dev/gomyadmin => ../../..
