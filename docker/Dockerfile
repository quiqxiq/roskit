FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api     ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker  ./cmd/worker

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /api     /api
COPY --from=builder /migrate /migrate
COPY --from=builder /worker  /worker

EXPOSE 8080

CMD ["/api"]
