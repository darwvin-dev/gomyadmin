FROM golang:1.26-bookworm AS runner

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .

EXPOSE 8080
CMD ["go", "run", "./templates/backend-go/cmd/server"]
