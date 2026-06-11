GO_PACKAGES := $(shell go list ./... | grep -Ev '/(examples|templates|tests/integration)(/|$$)')
YARN := npx --yes yarn@1.22.22
COVERAGE_THRESHOLD ?= 62

.PHONY: demo dev serve backend frontend test test-go test-coverage test-frontend vet doctor clean

demo:
	docker compose up --build

dev:
	docker compose up postgres

serve:
	go run ./cmd/gomyadmin serve

backend:
	go run ./templates/backend-go/cmd/server

frontend:
	cd templates/frontend-next-shadcn && npm run dev

test: test-go test-frontend

test-go:
	go test $(GO_PACKAGES)

test-coverage:
	go test $(GO_PACKAGES) -coverprofile=coverage.out -timeout 120s
	go tool cover -func=coverage.out | awk '/total:/ {pct=$$3+0; threshold=$(COVERAGE_THRESHOLD)+0; if (pct < threshold) { print "Coverage " $$3 " is below " threshold "%"; exit 1 } else { print "Coverage " $$3 " OK" }}'

vet:
	go vet $(GO_PACKAGES)

test-frontend:
	cd templates/frontend-next-shadcn && $(YARN) install --frozen-lockfile && $(YARN) run typecheck && $(YARN) run build

doctor:
	go run ./cmd/gomyadmin doctor

clean:
	docker compose down -v
