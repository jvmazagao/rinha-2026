FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache wget
COPY go.mod ./
RUN go mod download
COPY . .
RUN wget -q "https://raw.githubusercontent.com/zanfranceschi/rinha-de-backend-2026/main/resources/references.json.gz" -O resources/references.json.gz && \
    wget -q "https://raw.githubusercontent.com/zanfranceschi/rinha-de-backend-2026/main/resources/normalization.json" -O resources/normalization.json && \
    wget -q "https://raw.githubusercontent.com/zanfranceschi/rinha-de-backend-2026/main/resources/mcc_risk.json" -O resources/mcc_risk.json
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o api ./cmd/api

FROM scratch
WORKDIR /app
COPY --from=builder /app/api /app/api
COPY --from=builder /app/resources /app/resources
ENTRYPOINT ["/app/api"]
