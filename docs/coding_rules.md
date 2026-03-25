<code_editing_rules>
  <context>
    - Language: Go
    - Architecture: Clean Architecture（HTTP + stage workflow）
    - WebFramework: Gin
    - ORM: Gorm
  </context>

  <guiding_principles>
    - SOLID／SRP／YAGNI／DRY／KISS を尊重し、Controller は薄く保ち、複雑なオーケストレーションは application / workflow 層へ寄せる。
    - 共有ドメイン構造体を再利用する場合（`internal/common/domain.FetchedEmailDTO`, `internal/common/domain.ParsedEmail` など）は既存の JSON タグと不変条件を維持する。
    - 設定値は `oswrapper.OsWapperInterface` や `timewrapper.ClockInterface`, `crypto.Vault`, `ratelimit.Provider` など DI しやすい抽象を通じて取得する。
    - 新しい外部依存は `internal/library` に追加し、必要最小限の interface を公開して上位レイヤからは dig 経由で注入する。
    - 変更可能な package-level グローバル変数は禁止。`var Err...` のようなセンチネルエラーや定数は必要に応じて許容する。
    - トランザクションが必要な場合は `mysql.Transactional` を基本とし、既存コードが `db.Transaction(...)` や `db.Begin()` を使っているパッケージではスタイルを揃えて混在させない。
    - presentation / application 層は `oswrapper` や constructor 引数を通じて設定を受け取り、`internal/library` とサーバー起動前初期化以外で `os.Getenv` を直接呼ばない。
    - ログは必ず `logger.Interface` を使い、`log.With(logger.Component("..."))` でコンポーネント名を付与する。
    - ログへメールアドレス、トークン、Cookie などの秘匿情報をそのまま出さない。
    - `panic` は起動前初期化や不変条件を守るための最終手段に限定し、ビジネスロジックは `error` で返却する。
    - テストは独立性を保ち、既存の `t.Parallel()` 方針や Red→Green→Refactor を意識する。
  </guiding_principles>

  <naming_conventions>
    <common>
      - 変数／関数は camelCase、公開識別子は PascalCase。
      - import alias は lowerCamelCase（例: `manualapp`, `mfdomain`, `maapp`, `vrapp`）。
      - 複数形は対象を明確にした自然な名前を使う（`connections`, `parsedEmails`, `resolvedItems`）。
      - 関数動詞は既存規約（`Authorize`, `Callback`, `ListConnections`, `Disconnect`, `Execute`, `FetchFacts`, `EnsureByPlan`, `SaveAllIfAbsent`）に揃える。
      - `internal/app/presentation/{feature}` のように機能別ディレクトリを切り、controller とその HTTP DTO を同じ feature 配下へ置く。
      - Controller の request / response 構造体は controller ファイル内に置く。HTTP の shape が domain DTO と完全一致する場合だけ既存 DTO を再利用してよい。
      - モック / stub はローカル規約に合わせてテストファイル内へ置き、依存を局所化する（例: `stubVendorResolutionRepository`, `mockMailFetchStage`）。
    </common>

    <internal/{domain}>
      - /application
        - 可能なら具象 struct を非公開にし、`UseCase` / `UseCaseInterface` / `Command` / `Result` を公開する。
        - コンストラクタは `NewXxx`（`NewUseCase`, `NewAuthUseCase`, `NewDirectVendorResolutionAdapter` 等）。
      - /domain
        - JSON タグは必要に応じて保持し、Gorm タグは infrastructure モデル側へ置く。
        - 共有モデルは alias か再エクスポートで再利用し、重複定義を避ける。
      - /infrastructure
        - 構造体名は `XxxRepository`, `XxxAdapter`, `XxxFactory`, `XxxBuilder` を基本とし、コンストラクタは `NewXxx`。
        - メソッド名は `FindBy...`, `Save...`, `List...`, `Fetch...`, `Ensure...` など既存命名に従う。
    </internal/{domain}>

    <library>
      - クライアント構造体は基本 `Client`（または `{Lib}Wrapper`）とし、`New(...)` で生成する。
      - 上位レイヤからモック化が必要な場合は `ClientInterface`, `Limiter` などの interface をエクスポートする。
    </library>
  </naming_conventions>

  <implementation_rules>
    <doc_comments>
      - パッケージ外から利用される公開 struct／関数／メソッド（controller, usecase, repository, adapter 等）には目的・振る舞いが分かるドックコメントを付ける。private helper は必要な場合のみ記載する。
    </doc_comments>

    <error_handling>
      - 想定される失敗は型付きエラー（`application.ErrInvalidCredentials`, `mailaccountconnection/domain.ErrCredentialNotFound`, `mailfetch/domain.ErrConnectionNotFound` 等）で返し、`fmt.Errorf("component: %w", err)` で原因をラップする。
      - ランタイムで `panic` しない。`error` を返して呼び出し元（controller 等）で HTTP ステータスにマッピングする。
      - Controller / Middleware でエラーからレスポンスを組み立てる実装を統一的に保つ。
      - partial failure を扱う stage は、業務上の unresolved と technical failure を混同しない。
    </error_handling>

    <logging>
      - すべて `internal/library/logger` 経由でログ出力し、`request_id`, `user_id`, `component` など調査に必要なフィールドを構造化して付与する。
      - メールアドレス、トークン、Cookie、Authorization ヘッダなどの秘匿情報は raw で出さない。
    </logging>

    <context_handling>
      - application / workflow の公開メソッドは先頭に `context.Context` を受け取る。
      - `manualmailworkflow` は background 実行を持つため、HTTP リクエスト `context.Context` を長寿命 goroutine にそのまま引き回さず、新しい context を作って `request_id` / `user_id` / `job_id` など必要最小限の相関情報だけを引き継ぐ。
      - `context.WithoutCancel` は明確な要件がない限り導入せず、background workflow は専用 context で管理する。
    </context_handling>

    <transactions>
      - 共有 `*gorm.DB` からトランザクションを開始する場合は `mysql.Transactional` か `db.Transaction(...)` を使い、同一パッケージ内でスタイルを混在させない。
      - upsert / idempotent save は DB 制約と `OnConflict` を利用し、アプリ側の楽観に依存しない。
    </transactions>

    <env_config>
      - ランタイム設定は `oswrapper.OsWapperInterface` や dedicated constructor から取得する。
      - 起動前初期化（`internal/app/server/server.go`）や `internal/library` は、必要に応じて環境変数や secret client を直接扱ってよい。
    </env_config>
  </implementation_rules>

  <testing_rules>
    - テストファイル名は `xxx_test.go`。
    - 関数名は Go 標準の `TestXxx` 形式、またはパッケージで既に採用されているアンダースコア形式に合わせる。
    - 既に `t.Parallel()` を使っているパッケージ（library, repository, usecase 等）では引き続き並列化する。Gin controller テストなど共有状態があるものは直列のままでよい。
    - 複数パターンを検証する場合はテーブルドリブンテストを推奨する。
    - MySQL を用いるテストは `mysql.CreateNewTestDB()` を使い、戻り値のクリーンアップ関数で DB を削除する。必要に応じてトランザクションで境界を切る。
    - 既に testify の `assert` / `require` / `mock` を使用しているパッケージではそれに従い、新規パッケージは標準ライブラリか testify のどちらかに統一する。
    - モック / フェイクはテストファイル内に配置し、依存を局所化する。
  </testing_rules>
</code_editing_rules>
