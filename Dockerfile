FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o api ./cmd/api

FROM scratch
COPY --from=builder /app/api /api
COPY --from=builder /app/resources /resources
ENTRYPOINT ["/api"]
