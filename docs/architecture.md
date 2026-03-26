# アーキテクチャ設計

## コンテキスト

- Language: Go
- アーキテクチャ: Clean Architecture（HTTP + async workflow）
- WebFramework: Gin
- ORM: Gorm
- マイグレーション: Atlas
- DI コンテナ: `go.uber.org/dig`
- システム特性: HTTP API と非同期処理を併用する

## 関連ドキュメント

- HTTP API の詳細ルールは [API 共通仕様 設計](./api_design.md) を参照
- ログの詳細ルールは [ログ設計ルール](./log_rules.md) を参照
- ドメイン整理は `docs/ddd/**` を参照

## ディレクトリ構成

```text
cmd/
  app/
    main.go                         # エントリーポイント -> internal/app/server.Run

internal/
  app/
    server/                         # Gin 起動、共通依存初期化、CORS と middleware 登録
    router/                         # HTTP ルーティングと dig からの依存解決
    middleware/                     # RequestID, request summary, panic recovery, JWT 認証
    httpresponse/                   # 標準エラーレスポンス
    presentation/
      auth/                         # 認証系 controller と HTTP DTO
      billing/                      # Billing 一覧 / Monthly Trend / Month Detail controller と HTTP DTO
      mailaccountconnection/
      manualmailworkflow/
  di/                               # dig モジュール群
  library/                          # 共通ラッパー: logger, mysql, gmail/gmailService, openai, oswrapper, ratelimit, secret, sendMailer, crypto, timewrapper
  auth/                             # 認証ドメイン（domain/application/infrastructure）
  billing/                          # Billing ドメイン
  billingquery/                     # Billing read API
  common/                           # 共有ドメインモデル（Email, ParsedEmail, Billing, Vendor, VendorResolutionPolicy 等）
  mailaccountconnection/            # Gmail OAuth 連携
  mailfetch/                        # メール取得
  mailanalysis/                     # AI 解析
  vendorresolution/                 # Vendor 解決
  billingeligibility/               # Billing 成立可否判定
  manualmailworkflow/               # 非同期 workflow
```

`internal/di` には `auth.go`, `mail_account_connection.go`, `mailfetch.go`, `mailanalysis.go`, `vendorresolution.go`, `billingeligibility.go`, `billing.go`, `manualmailworkflow.go`, `presentation.go`, `dig.go` などの dig モジュールを配置する。

## レイヤ構成

### Presentation 層

- 配置例:
  - `internal/app/presentation/{feature}`
  - `internal/app/middleware`
  - `internal/app/router`
  - `internal/app/server`
- 役割:
  - Gin のリクエストを受け、HTTP DTO へ bind / validate した上で application 層へ渡す
  - application 層の結果を HTTP ステータス / JSON レスポンスへ変換する
  - `RequestID`, `RequestSummary`, `Recovery`, `AuthMiddleware` により request-scoped 情報を `context.Context` へ載せる
- 依存可能:
  - application 層の interface / usecase
  - domain DTO
  - `internal/library/logger`
  - `internal/app/httpresponse`
- 禁止事項:
  - Gorm / Gmail / OpenAI など infrastructure 具体実装の直接利用

### Application 層

- 配置例:
  - `internal/{domain}/application`
  - `internal/billingeligibility/application`
  - `internal/manualmailworkflow/application`
- 役割:
  - ユースケース単位の入力検証、オーケストレーション、部分成功 / 部分失敗の集約を担当する
- 依存可能:
  - domain 層
  - application 層で定義した port / interface
  - 必要最小限の cross-cutting abstraction
- 禁止事項:
  - Gin / Gorm / 外部 SDK の具体実装への直接依存
  - HTTP 入出力や永続化詳細をそのまま持ち込むこと

### Domain 層

- 配置例:
  - `internal/auth/domain`
  - `internal/mailaccountconnection/domain`
  - `internal/mailfetch/domain`
  - `internal/mailanalysis/domain`
  - `internal/vendorresolution/domain`
  - `internal/billing/domain`
  - `internal/billingeligibility/domain`
  - `internal/common/domain`
- 役割:
  - ビジネスエンティティ、不変条件、値オブジェクト、policy を保持する
  - `internal/common/domain` には共有モデルと cross-stage policy を置く
- 代表例:
  - `Email`
  - `ParsedEmail`
  - `Billing`
  - `Vendor`
  - `VendorResolutionPolicy`
  - `BillingEligibility`
- 禁止事項:
  - Gin / Gorm / 外部 SDK への依存

### Infrastructure 層

- 配置例:
  - `internal/{domain}/infrastructure`
  - `internal/billingquery/infrastructure`
  - `internal/manualmailworkflow/infrastructure`
- 役割:
  - DB、外部 API、SDK、メッセージング、ファイル I/O など外部依存との接続を担当する
  - application 層が定義した port を実装する
- 依存可能:
  - Gorm / Gmail SDK / OpenAI SDK / `internal/library/**`
  - application 層が定義した port
- 禁止事項:
  - presentation 層への依存

## Library 層

### 目的

- 外部 SDK や cross-cutting concern を `internal/library` に閉じ込め、上位レイヤが SDK 直依存しないようにする

### 主なパッケージ

- `logger`: zap ベースの構造化ログ
- `mysql`: 接続生成、transaction helper、テスト DB 作成
- `gmail` / `gmailService`: Gmail API / OAuth loader
- `openai`: OpenAI クライアント
- `ratelimit`: Redis-backed limiter provider
- `crypto`: OAuth token などの暗号化 / digest
- `secret`
- `oswrapper`
- `sendMailer`
- `timewrapper`

### 方針

- server 初期化や `internal/library` 配下では必要に応じて環境変数や secret を直接扱ってよい
- application / presentation 層は interface 経由で受け取る

## Dependency Injection

### 基本方針

- 依存注入は `internal/di` に集約する
- `dig.go` が共通依存（mysql, gmail, openai, oswrapper, ratelimit, logger, vault, clock）を登録する
- 各機能モジュールが repository / adapter / usecase / controller を Provide する
