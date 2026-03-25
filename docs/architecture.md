<architecture_rules>
  <context>
    - Language: Go
    - アーキテクチャ: Clean Architecture（HTTP + async workflow）
    - WebFramework: Gin
    - ORM: Gorm
    - マイグレーション: Atlas
    - 主ワークフロー: `internal/manualmailworkflow`（`POST` は受付のみを返し、バックグラウンドで `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` を実行する）
    - DI コンテナ: go.uber.org/dig
  </context>

  <related_documents>
    - HTTP API の詳細ルールは [こちら](./api_design.md) を参照してください
    - ログの詳細ルールは [こちら](./log_rules.md) を参照してください
    - ドメイン整理は `docs/ddd/**` を参照してください
  </related_documents>

  <directory_structure>
    <root_packages>
      - cmd/
        - app/
          - main.go               // エントリーポイント → internal/app/server.Run
      - internal/
        - app/
          - server/               // Gin 起動、共通依存初期化、CORS と middleware 登録
          - router/               // HTTP ルーティングと dig からの依存解決
          - middleware/           // RequestID, request summary, panic recovery, JWT 認証
          - httpresponse/         // 標準エラーレスポンス
          - presentation/
            - auth/               // 認証系 controller と HTTP DTO
            - mailaccountconnection/
            - manualmailworkflow/
        - di/                     // dig モジュール（auth.go, mail_account_connection.go, mailfetch.go, mailanalysis.go, vendorresolution.go, billingeligibility.go, billing.go, manualmailworkflow.go, presentation.go, dig.go）
        - library/                // 共通ラッパー: logger, mysql, gmail/gmailService, openai, oswrapper, ratelimit, secret, sendMailer, crypto, timewrapper
        - auth/                   // 認証ドメイン（domain/application/infrastructure）
        - billing/                // Billing 生成・重複制御・保存 stage
        - common/                 // 共有ドメインモデル（Email, ParsedEmail, Billing, Vendor, VendorResolutionPolicy 等）
        - mailaccountconnection/  // Gmail OAuth 連携と資格情報管理
        - mailfetch/              // メール取得 stage
        - mailanalysis/           // AI 解析 stage
        - vendorresolution/       // canonical Vendor 解決 stage
        - billingeligibility/     // Billing 成立可否判定 stage
        - manualmailworkflow/     // 非同期 workflow の受付と実行を束ねる
    </root_packages>
  </directory_structure>

  <layers>
    <presentation>
      - 配置:
        - `internal/app/presentation/{feature}`
        - `internal/app/middleware`
        - `internal/app/router`
        - `internal/app/server`
      - 役割:
        - Gin のリクエストを受け、HTTP DTO へ bind / validate した上で application 層へ渡す。
        - application 層の結果を HTTP ステータス / JSON レスポンスへ変換する。
        - `RequestID`, `RequestSummary`, `Recovery`, `AuthMiddleware` により request-scoped 情報を `context.Context` へ載せる。
      - 依存可能:
        - application 層の interface / usecase
        - domain DTO
        - `internal/library/logger`
        - `internal/app/httpresponse`
      - 禁止事項:
        - Gorm / Gmail / OpenAI など infrastructure 具体実装の直接利用
    </presentation>

    <application>
      - 配置:
        - `internal/{domain}/application`
        - `internal/billingeligibility/application`
        - `internal/manualmailworkflow/application`
      - 役割:
        - ユースケース単位の入力検証、オーケストレーション、部分成功 / 部分失敗の集約を担当する。
      - 代表例:
          - `auth/application`: register / login / refresh / logout / verify email
          - `mailaccountconnection/application`: Gmail OAuth state 管理、資格情報保存、一覧 / 解除
          - `mailfetch/application`: 利用可能な連携解決、provider fetch、メール保存
          - `mailanalysis/application`: OpenAI 解析、`ParsedEmail` 永続化
          - `vendorresolution/application`: alias lookup、必要なら canonical Vendor 自動登録
          - `billingeligibility/application`: `ParsedEmail` と解決済み Vendor から Billing 成立可否を評価
          - `billing/application`: eligible item から Billing 生成、idempotent 保存、created / duplicate / failure 集約
          - `manualmailworkflow/application`: workflow 受付、background 実行、履歴集約
      - 注意点:
        - `POST /api/v1/manual-mail-workflows` は短時間で `202 Accepted` を返し、実処理はバックグラウンドで進める。
        - workflow 履歴は `manual_mail_workflow_histories` と `manual_mail_workflow_stage_failures` に保存し、header には件数、child table には失敗理由を保持する。
        - stage の元 result では technical failure と unresolved / ineligible / duplicate を分けて扱うが、workflow 履歴ではユーザー表示用の failure count / reason_code 側に集約する。
    </application>

    <domain>
      - 配置:
        - `internal/auth/domain`
        - `internal/mailaccountconnection/domain`
        - `internal/mailfetch/domain`
        - `internal/mailanalysis/domain`
        - `internal/vendorresolution/domain`
        - `internal/billing/domain`
        - `internal/billingeligibility/domain`
        - `internal/common/domain`
      - 役割:
        - ビジネスエンティティ、不変条件、値オブジェクト、policy を保持する。
        - `internal/common/domain` には共有モデルと cross-stage policy を置く。
          - `Email`, `ParsedEmail`, `Billing`, `Vendor`
          - `VendorResolutionPolicy`, `BillingEligibility`
      - 禁止事項:
        - Gin / Gorm / 外部 SDK への依存
    </domain>

    <infrastructure>
      - 配置:
        - `internal/{domain}/infrastructure`
        - `internal/manualmailworkflow/infrastructure`
      - 役割:
        - Gorm repository、OAuth exchanger、Gmail profile fetcher、Gmail session builder、OpenAI analyzer adapter など外部依存との接続を担当する。
        - `manualmailworkflow/infrastructure` は各 stage usecase を直接呼び出す adapter を持つ。
        - `vendorresolution/infrastructure` は `vendors` / `vendor_aliases` の read / write に責務を限定する。
      - 依存可能:
        - Gorm / Gmail SDK / OpenAI SDK / `internal/library/**`
        - application 層が定義した port
      - 禁止事項:
        - presentation 層への依存
    </infrastructure>
  </layers>

  <workflow>
    - 入口:
      - `POST /api/v1/manual-mail-workflows`
    - 実行順:
      - start usecase が workflow を受け付け、dispatcher が background 実行へ渡す。
      - background runner が `mailfetch`, `mailanalysis`, `vendorresolution`, `billingeligibility`, `billing` の各 stage を順に呼ぶ。
    - `mailfetch`:
      - 利用可能な `MailAccountConnection` を解決する。
      - Gmail セッションを生成して provider からメールを取得する。
      - `emails` テーブルへメタデータを idempotent に保存する。
      - 取得した本文は background workflow 内の in-memory payload として `mailanalysis` に渡す。
    - `mailanalysis`:
      - OpenAI analyzer で本文を解析し、`parsed_emails` を保存する。
      - 保存済み `ParsedEmail` と source email の必要メタデータを workflow へ返す。
    - `vendorresolution`:
      - `ParsedEmail.vendorName` と source email の `subject` / `from` / `to` をもとに canonical Vendor を決定する。
      - unresolved なら candidate vendor 名から `vendors` と `name_exact` alias の自動補完を試す。
      - unresolved は業務結果として返し、technical failure とは分離する。
    - `billingeligibility`:
      - 解決済み Vendor と `ParsedEmail` をもとに Billing 成立可否を評価する。
      - `billing_date` は任意として扱い、必須項目不足や不正値は `ineligible` または `failure` に分類する。
    - `billing`:
      - `BillingEligibility` で成立した対象から `Billing` を生成し、idempotent に保存する。
    - 履歴 / 状態参照:
      - `manual_mail_workflow_histories` が `workflow_id` ごとの受付条件、状態、stage ごとの成功件数 / 失敗件数を保持する。
      - `manual_mail_workflow_stage_failures` が `stage` / `reason_code` / `external_message_id` を保持し、将来の `GET /api/v1/manual-mail-workflows/:workflow_id` は両テーブルから状態を返す前提で設計する。
  </workflow>

  <context_management>
    - HTTP request の `context.Context` は workflow 受付までに使い、background 実行では新しい `context.Context` を作る。
    - `RequestID` middleware が `request_id` を `gin.Context` と `context.Context` の両方へ設定する。
    - `AuthMiddleware` が認証済み `user_id` を `gin.Context` と `context.Context` に設定する。
    - background workflow では `job_id` と `user_id` を context に積み、logger は `WithContext(ctx)` により `request_id` / `job_id` / `user_id` を自動付与する。
  </context_management>

  <library_layer>
    - 目的:
      - 外部 SDK や cross-cutting concern を `internal/library` に閉じ込め、上位レイヤが SDK 直依存しないようにする。
    - 主なパッケージ:
      - `logger`: zap ベースの構造化ログ
      - `mysql`: 接続生成、transaction helper、テスト DB 作成
      - `gmail` / `gmailService`: Gmail API / OAuth loader
      - `openai`: OpenAI クライアント
      - `ratelimit`: Redis-backed limiter provider
      - `crypto`: OAuth token などの暗号化 / digest
      - `secret`, `oswrapper`, `sendMailer`, `timewrapper`
    - 方針:
      - server 初期化や `internal/library` 配下では必要に応じて環境変数や secret を直接扱ってよい。
      - application / presentation 層は interface 経由で受け取る。
  </library_layer>

  <dependency_injection>
    - 依存注入は `internal/di` に集約する。
      - `dig.go` が共通依存（mysql, gmail, openai, oswrapper, ratelimit, logger, vault, clock）を登録する。
      - 各機能モジュールが repository / adapter / usecase / controller を Provide する。
      - `manualmailworkflow` は direct stage adapter に加え、dispatcher / start usecase を束ねる。
    - ブートストラップ手順:
      - `cmd/app/main.go` → `internal/app/server.Run`
      - `server.Run` が secret client、logger、MySQL、rate limit provider、Gmail / OpenAI client、Vault を初期化
      - `di.BuildContainer` が container を構築
      - `router.Router` が controller / middleware を解決して route を登録
  </dependency_injection>
</architecture_rules>
