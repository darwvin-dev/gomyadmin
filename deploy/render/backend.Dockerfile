FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gomyadmin-demo-api ./templates/backend-go/cmd/server

FROM debian:bookworm-slim AS runner

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV PORT=8080
COPY --from=builder /out/gomyadmin-demo-api /usr/local/bin/gomyadmin-demo-api

EXPOSE 8080
CMD ["/usr/local/bin/gomyadmin-demo-api"]
