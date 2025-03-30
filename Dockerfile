FROM golang:1.23.4-alpine3.19 AS builder

RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    openssl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -o /app/bot

FROM alpine:3.19

RUN apk add --no-cache \
    ca-certificates \
    openssl

WORKDIR /app

COPY --from=builder /app/bot /app/bot

ENTRYPOINT ["/app/bot"]