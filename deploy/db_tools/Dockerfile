FROM golang:1.24-alpine

WORKDIR /app

RUN apk add --no-cache ca-certificates curl git tzdata && \
    update-ca-certificates

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
RUN curl -sSf https://atlasgo.sh | sh -s -- -y && atlas version

COPY internal ./internal
COPY tools ./tools
COPY atlas.hcl ./atlas.hcl
