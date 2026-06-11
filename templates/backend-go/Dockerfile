FROM golang:1.25-bookworm AS runner

WORKDIR /app
COPY . .
RUN go mod tidy

EXPOSE 8080
CMD ["go", "run", "./templates/backend-go/cmd/server"]
