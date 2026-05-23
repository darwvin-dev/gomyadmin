.PHONY: demo dev serve backend frontend test test-go test-frontend vet doctor clean

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
	go test ./...

vet:
	go vet ./...

test-frontend:
	cd templates/frontend-next-shadcn && npm install && npm run typecheck && npm run build

doctor:
	go run ./cmd/gomyadmin doctor

clean:
	docker compose down -v
