ARG GO_VERSION=1.25.2
ARG APP_ROOT=/home/dev/backend

# ==================================================
# builder: 本番ビルド用
# ==================================================
FROM golang:${GO_VERSION}-alpine3.22 AS builder

RUN apk add --no-cache \
    build-base \
    ca-certificates \
    git \
    tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -buildvcs=false -o /usr/local/bin/app ./cmd/app

# ==================================================
# runtime: 本番/デプロイ用イメージ (軽量)
# ==================================================
FROM alpine:3.22 AS runtime

ARG APP_ROOT
ARG APP_USER=dev

ENV TZ=Asia/Tokyo \
    LANG=ja_JP.UTF-8 \
    LANGUAGE=ja_JP:ja \
    LC_ALL=ja_JP.UTF-8 \
    MUSL_LOCPATH=/usr/share/i18n/locales/musl \
    RATE_LIMIT_SCRIPT_PATH=${APP_ROOT}/internal/library/redis/script/scripts/rate_limit.lua

RUN apk add --no-cache \
    ca-certificates \
    curl \
    musl-locales \
    musl-locales-lang \
    tzdata && \
    cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone && \
    update-ca-certificates

RUN addgroup -g 1000 ${APP_USER} && \
    adduser -D -u 1000 -G ${APP_USER} -s /bin/sh ${APP_USER} && \
    mkdir -p ${APP_ROOT} && \
    chown -R ${APP_USER}:${APP_USER} ${APP_ROOT}

WORKDIR ${APP_ROOT}
COPY --from=builder /app/internal/library/redis/script/scripts ${APP_ROOT}/internal/library/redis/script/scripts
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

USER ${APP_USER}
CMD ["app"]
