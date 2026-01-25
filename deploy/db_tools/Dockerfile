FROM golang:1.24-alpine

WORKDIR /app

RUN apk add --no-cache ca-certificates git tzdata && \
    update-ca-certificates

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY tools ./tools
