ARG GO_VERSION=1.25.2
ARG APP_ROOT=/home/dev/backend

# ==================================================
# builder: 本番ビルド用
# ==================================================
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -buildvcs=false -o /usr/local/bin/app ./cmd/app

# ==================================================
# runtime: 本番/デプロイ用イメージ (軽量)
# ==================================================
FROM debian:bookworm-slim AS runtime

ARG APP_ROOT
ARG APP_USER=dev

ENV TZ=Asia/Tokyo \
    LANG=ja_JP.UTF-8 \
    LANGUAGE=ja_JP:ja \
    LC_ALL=ja_JP.UTF-8 \
    RATE_LIMIT_SCRIPT_PATH=${APP_ROOT}/internal/library/redis/script/scripts/rate_limit.lua

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    locales \
    curl && \
    localedef -i ja_JP -c -f UTF-8 -A /usr/share/locale/locale.alias ja_JP.UTF-8 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN groupadd -g 1000 ${APP_USER} && \
    useradd -m -u 1000 -g ${APP_USER} -s /bin/bash ${APP_USER} && \
    mkdir -p ${APP_ROOT} && \
    chown -R ${APP_USER}:${APP_USER} ${APP_ROOT}

WORKDIR ${APP_ROOT}
COPY --from=builder /app/internal/library/redis/script/scripts ${APP_ROOT}/internal/library/redis/script/scripts
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

USER ${APP_USER}
CMD ["app"]
