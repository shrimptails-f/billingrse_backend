<code_editing_rules>
  <context>
    - Language: Go
    - Architecture: Clean Architecture（HTTP + メッセージングパイプライン）
    - WebFramework: Gin
    - ORM: Gorm
  </context>

  <guiding_principles>
    - SOLID／SRP／YAGNI／DRY／KISS を尊重し、Controller は薄く保ち、複雑なオーケストレーションは application 層へ寄せる。
    - ドメイン構造体が API レスポンスや保存データを兼ねる場合（`internal/emailstore/domain.Email`, `internal/common/domain.FetchedEmailDTO` など）は既存の JSON タグを維持する。
    - 設定値は `oswrapper.OsWapperInterface` や `timewrapper.ClockInterface`, `crypto.Vault`, `ratelimit.Provider` など DI しやすい抽象を通じて取得する。
    - 新しい外部依存は `internal/library` に追加し、必要最小限の interface を公開して上位レイヤからは dig 経由で注入する。
    - 変更可能な package-level グローバル変数は禁止。`var Err...` のようなセンチネルエラーや定数は必要に応じて許容する。
    - トランザクションが必要な場合は `mysql.Transactional` を基本とし、既存コードが `db.Begin()` を直接使っているパッケージではスタイルを合わせて混在させない。
    - プレゼンテーション／アプリケーション／messaging 層は `oswrapper` を通じて環境変数を読む。`internal/library` 配下のみ `os.Getenv` を直接使用してよい。
    - ログは必ず `logger.Interface` を使い、`log.With(logger.String("component", ...))` でコンポーネント名を付与する。
    - `panic` は初期化失敗や不変条件を守るための最終手段（例: ロガー初期化、SMTP クライアント DI、mysql.Transactional の Begin 失敗）に限定し、ビジネスロジックは `error` で返却する。
    - テストは独立性を保ち、既存の `t.Parallel()` 方針や Red→Green→Refactor を意識する。
  </guiding_principles>

  <naming_conventions>
    <common>
      - 変数／関数は camelCase、公開識別子は PascalCase。
      - import alias は lowerCamelCase（例: `emailanalysisapp`, `gc`, `gs`, `cd`）。
      - 複数形は `List` を付与（`agentList`, `ListAgentsResponse`）。
      - 関数動詞は既存規約（`GetXxx`, `SaveXxx`, `UpdateXxx`, `ListXxx`, `CreateXxx`）に揃える。
      - interface 名はパッケージ既存の命名を踏襲しつつ、移行可能であれば `FooUseCase`（interface）＋`fooUseCase`（具象 struct）形式へ段階的に寄せてもよい。既存コードの互換性を壊さない範囲で徐々に統一する。
      - Controller のリクエスト/レスポンス構造体はハンドラファイル内に置く。ただし、HTTP の形がドメイン DTO と完全に一致する場合（例: Agent API が `domain.ListAgentsResponse` をそのまま返す）は既存 DTO を再利用してよい。差異がある場合のみローカル struct を定義し、JSON タグを API に合わせる。
      - モックはローカル規約（例: `mockEmailCredentialUsecase`, `MockAgentUsecase`）を守り、既に testify/mock を使っているパッケージではそれに従う。
    </common>

    <internal/{domain}>
      - /application
        - 可能なら具象 struct を非公開にし、`UseCase` / `Usecase` / `...Interface` を公開する。
        - コンストラクタは `NewXxx`（`NewAuthUseCase`, `NewAgentUsecase`, `NewSessionFactory` 等）。
      - /domain
        - JSON タグは必要に応じて保持し、Gorm タグは infrastructure モデル側へ。
        - 共有モデルは alias で再利用し、重複定義を避ける。
      - /infrastructure
        - 構造体名は `XxxRepository`, `XxxAdapter`, `XxxFactory`、コンストラクタは `NewXxx`。
        - メソッド名は `FindBy...`, `Save...`, `List...` など既存命名に従う。
    </internal/{domain}>

    <library>
      - クライアント構造体は基本 `Client`（または `{Lib}Wrapper`）とし、`New(...)` で生成する。
      - 上位レイヤからモック化が必要な場合は `ClientInterface`, `Limiter` などの interface をエクスポートする。
    </library>
  </naming_conventions>

  <implementation_rules>

    <doc_comments>
      - パッケージ外から利用される公開 struct／関数／メソッド（controller, usecase, repository, adapter 等）には目的・振る舞いが分かるドックコメントを付ける。プライベートヘルパーは必要な場合のみ記載。
    </doc_comments>

    <error_handling>
      - 想定される失敗は型付きエラー（`application.ErrInvalidCredentials`, `emailanalysisapp.ErrCredentialNotFound` 等）で返し、`fmt.Errorf("component: %w", err)` で原因をラップする。
      - ランタイムで `panic` しない。`error` を返して呼び出し元（Controller 等）で HTTP ステータスにマッピングする。
      - Controller / Middleware でエラーからレスポンスを組み立てる実装を統一的に保つ。
    </error_handling>

    <logging>
      - すべて `internal/library/logger` 経由でログ出力し、ユーザー ID や Gmail ID など調査に必要なフィールドを構造化して付与する。
    </logging>

    <context_handling>
      - application / messaging の公開メソッドは先頭に `context.Context` を受け取る。HTTP リクエストより長生きさせたい処理のみ `context.WithoutCancel` を利用する（emailanalysis 参照）。
    </context_handling>

    <transactions>
      - 共有 `*gorm.DB` からトランザクションを開始する場合は `mysql.Transactional` を使用し、`defer cleanUp()` で確実に終了させる。
      - 既存コードが `db.Begin()` を直接使っているパッケージではそのスタイルに合わせ、同一パッケージ内で混在させない。
    </transactions>

    <env_config>
      - ランタイム設定は `oswrapper.OsWapperInterface` から取得し、`internal/di/auth.go` の `parseTokenTTL` のように妥当なデフォルトを設ける。
      - `internal/library` パッケージのみ `os.Getenv` を直接利用してよい。
    </env_config>

  </implementation_rules>

  <testing_rules>
    - テストファイル名は `xxx_test.go`。
    - 関数名は Go 標準の `TestXxx` 形式、またはパッケージで既に採用されているアンダースコア形式（例: `Test_LoginHandler_InvalidCredentials`）に合わせる。
    - 既に `t.Parallel()` を使っているパッケージ（library や repository, usecase 等）では引き続き並列化する。Gin Handler テストなど状態共有があるものは直列のままでよい。
    - 複数パターンを検証する場合はテーブルドリブンテストを推奨。
    - MySQL を用いるテストは `mysql.CreateNewTestDB()` を使い、戻り値のクリーンアップ関数で DB を削除する。必要に応じて `mysql.Transactional` でトランザクションを張る。
    - 既に testify の `assert` / `require` / `mock` を使用しているパッケージではそれに従い、新規パッケージは標準ライブラリ or testify いずれかを選んで統一する。
    - モック/フェイクはテストファイル内に配置し（例: `mockEmailAnalysisUseCase`）、依存を局所化する。
  </testing_rules>
</code_editing_rules>
