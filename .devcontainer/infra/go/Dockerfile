FROM golang:1.25.2-alpine3.22

ARG APP_ROOT

ENV TZ=Asia/Tokyo \
    LANG=ja_JP.UTF-8 \
    LANGUAGE=ja_JP:ja \
    LC_ALL=ja_JP.UTF-8 \
    MUSL_LOCPATH=/usr/share/i18n/locales/musl

# 共通パッケージをまとめてインストール
RUN apk add --no-cache \
      bash \
      build-base \
      ca-certificates \
      curl \
      git \
      gnupg \
      keychain \
      musl-locales \
      musl-locales-lang \
      nodejs \
      npm \
      python3 \
      redis \
      ripgrep \
      sudo \
      tzdata \
      vim && \
    cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone && \
    update-ca-certificates && \
    arch="$(apk --print-arch)" && \
    case "${arch}" in \
      x86_64) task_arch='amd64' ;; \
      aarch64) task_arch='arm64' ;; \
      *) echo "unsupported architecture: ${arch}" >&2; exit 1 ;; \
    esac && \
    curl -sL "https://github.com/go-task/task/releases/download/v3.9.0/task_linux_${task_arch}.tar.gz" | tar -xz -C /usr/local/bin task

RUN curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/usr/local/bin sh

ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt \
    REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt \
    NO_PROXY=localhost,127.0.0.1,::1,auth.openai.com

# Go ツール (root でインストール -> 全ユーザーで共有)
RUN go install github.com/air-verse/air@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install golang.org/x/tools/gopls@latest && \
    go install github.com/fatih/gomodifytags@latest && \
    go install github.com/rinchsan/gosimports/cmd/gosimports@latest  && \
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3

# 開発ユーザー作成
RUN addgroup -g 1000 dev && \
    adduser -D -u 1000 -G dev -s /bin/bash dev && \
    chown -R dev:dev /home/dev && \
    chown -R dev:dev /go/pkg/

# VS Code Server 用ディレクトリ
# VS Code Server 用のディレクトリを事前に作成して権限を設定
RUN mkdir -p /home/dev/.vscode-server/bin /home/dev/.vscode-server/extensions && \
    chown -R dev:dev /home/dev/.vscode-server
RUN mkdir -p /home/dev/.codex && chown -R dev:dev /home/dev/.codex

# Atlasインストール(マイグレーションのバージョン管理)
RUN curl -sSf https://atlasgo.sh | sh

WORKDIR $APP_ROOT

USER dev

# dev ユーザー配下に npm グローバルインストール（sudo不要）
ENV NPM_CONFIG_PREFIX=/home/dev/.local
RUN mkdir -p /home/dev/.local && npm i -g @openai/codex@latest

# bashrc 設定
COPY --chown=dev:dev .devcontainer/infra/go/.bashrc /home/dev/.bashrc
COPY --chown=dev:dev .devcontainer/infra/go/config.toml /home/dev/.codex/config.toml

# Claude Code インストール
RUN curl -fsSL https://claude.ai/install.sh | bash && \
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/dev/.bashrc && \
    echo 'eval `keychain --eval --agents ssh id_rsa 2>/dev/null`' >> /home/dev/.bashrc

ENV PATH="/home/dev/.local/bin:${PATH}"
