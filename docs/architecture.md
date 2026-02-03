<architecture_rules>
  <context>
    - Language: Go
    - アーキテクチャ: Clean Architecture（HTTP + メッセージングパイプライン）
    - WebFramework: Gin
    - ORM: Gorm
    - マイグレーション: Atlas
    - 非同期パイプライン: internal/messaging（Gmail / OpenAI / emailstore）
    - DI コンテナ: go.uber.org/dig
  </context>

  <directory_structure>
    <root_packages>
      - cmd/
        - app/
          - main.go               // エントリーポイント → internal/app/server.Run
      - internal/
        - app/
          - server/               // Gin 起動、環境変数・MySQL・レートリミット・Gmail/OpenAI クライアントなどの初期化
          - router/               // HTTP ルーティングと dig からの依存解決
          - middleware/           // JWT 認証や共通ミドルウェア
          - presentation/         // Controller・HTTP DTO・ローカル interface・Gin ベースのテスト
        - di/                     // dig モジュール（auth.go, agent.go, emailstore.go, messaging.go, presentation.go 等）
        - library/                // 共通ラッパー: logger, mysql, gmail/gmailService, openai, oswrapper, ratelimit, redis, retry, sendMailerClient, crypto, timewrapper
        - auth/                   // 認証ドメイン（domain/application/infrastructure）
        - agent/                  // AI エージェントトークン（domain/application/infrastructure）
        - emailcredential/        // Gmail OAuth 資格情報管理
        - emailstore/             // 解析済みメールの永続化
        - common/                 // 共通 DTO とメール分析ログ
        - emailanalysis/          // メッセージングパイプラインを呼び出すアプリケーション層
        - messaging/
          - domain/               // パイプラインのポート・JobParams・PipelineService 実装
          - application/          // PipelineFactory や Dispatcher のヘルパー
          - infrastructure/       // Gmail セッションアダプタ / OpenAI アナライザ / ロギング・emailstore アダプタ
    </root_packages>
  </directory_structure>

  <layers>
    <presentation>
      - 配置:
        - internal/app/presentation（コントローラと HTTP DTO）
        - internal/app/middleware（JWT 検証・ユーザー情報注入）
        - internal/app/router（ルート登録と dig 解決）
        - internal/app/server（プロセス起動）
      - 役割:
        - Gin のリクエストをアプリケーション層 DTO に変換し、バリデーションを行った上で 1 Handler = 1 UseCase を呼び出す。
        - 受け取ったエラーを HTTP ステータスへマッピングし、logger.Interface で構造化ログを出力する。
        - 具体型ではなく振る舞いに依存したい場合は Controller 内で小さな interface を定義する（例: AgentUsecase）。
        - ミドルウェアは oswrapper 経由で JWT 秘密鍵を取得し、トークン検証後に `userID` を Gin コンテキストへ保存する。
      - 依存可能なレイヤ:
        - application 層インターフェース、domain DTO、ミドルウェアのヘルパー、logger.Interface
      - 禁止事項:
        - Gorm や mysql 接続、Gmail/OpenAI クライアントなどの infrastructure 直接利用
    </presentation>

    <application>
      - 配置:
        - internal/{domain}/application（auth, agent, emailcredential, emailstore, common/email-analysis-log 等）
        - internal/emailanalysis/application（キュー投入ユースケース）
        - internal/messaging/application（パイプラインの Factory / Dispatcher）
      - 役割:
        - 各ユースケースの interface（`AuthUseCaseInterface`, `AgentUsecase`, `emailanalysisapp.UseCase` など）を保持し、オーケストレーション・バリデーション・トランザクション管理を実装する。
        - リポジトリや他コンテキストと連携する：
          - `internal/agent/application` は複数テーブルを更新するため明示的に `*gorm.DB` トランザクションを管理。
          - `internal/emailcredential/application` は `crypto.Vault` と oswrapper を使って OAuth トークンを暗号化/復号。
          - `internal/emailanalysis/application` は messaging ドメインを用いた事前チェックを行い、`PipelineService` を非同期実行。
          - `internal/common/application` はメール分析ログ操作を提供し、messaging の logging adapter から呼び出される。
        - DI された `oswrapper`, `timewrapper.ClockInterface`, `crypto.Vault`, `logger.Interface` などのヘルパー、および必要に応じて他アプリケーション層サービス（emailstore.UseCase など）へ依存する。
      - 注意点:
        - パッケージ内で歴史的に `*gorm.DB` を受け取っている場合（agent usecase）はそれに従い、グローバル変数は作らない。
        - 長時間処理（メール分析）はバックグラウンド goroutine で実行し、`context.Context` を通じてキャンセル情報を伝播させる。
    </application>

    <domain>
      - 配置:
        - internal/{domain}/domain
        - internal/common/domain（共有 DTO・エージェント定義・暗号ヘルパー）
        - internal/messaging/domain（パイプライン用エンティティとポート）
      - 役割:
        - `auth/domain.User`, `emailstore/domain.Email`, `common/domain.FetchedEmailDTO`, `messaging/domain.AnalysisResult` などビジネスエンティティとドメインロジックを保持する。
        - 共有モデルが必要な場合は再エクスポート（agent ドメインが common/domain.Agent を alias する等）で重複を避ける。
        - API レスポンスや永続化結果としてシリアライズする構造体は JSON タグを保持し、Gorm タグなどインフラ固有の情報は infrastructure 層に置く。
      - 禁止事項:
        - Gin / Gorm / 外部 SDK への依存
    </domain>

    <infrastructure>
      - 配置:
        - internal/{domain}/infrastructure（Gorm リポジトリ、SMTP メーラ、トークンリフレッシュなど）
        - internal/messaging/infrastructure（Gmail アダプタ、OpenAI アナライザ、emailstore/logging アダプタ）
      - 役割:
        - ドメインエンティティとデータベースモデルを相互変換し、外部サービス呼び出しを `internal/library` のクライアント経由で行う。
        - 既存ユースケースをアダプタ越しに再利用する（ResultSaverAdapter が emailstore.UseCase を呼ぶ、LoggingAdapter が common/application を呼ぶ）。
      - 依存可能:
        - Gorm / Google SDK / OpenAI SDK / redis, oswrapper, logger, 再利用を許可したアプリケーション層
      - 禁止事項:
        - presentation 層への依存
    </infrastructure>
  </layers>

  <messaging>
    - Domain (`internal/messaging/domain`):
      - `JobParams`・正規化された `Message`/`AnalysisResult`・資格情報オブジェクト・`PipelineService` を定義し、`SessionFactoryPort` / `AnalyzerFactoryPort` / `ResultSaverPort` / `AnalysisLogPort` を束ねる。
    - Application (`internal/messaging/application`):
      - `PipelineFactory` や Dispatcher を提供し、DI 層がプロバイダ/アナライザのファクトリを登録しやすくする。
    - Infrastructure (`internal/messaging/infrastructure`):
      - Gmail credential provider が emailcredential リポジトリと crypto.Vault を用いてトークンを復号。
      - Session factory が GmailService, Gmail REST ラッパー, emailstore.UseCase, token refresher, oswrapper の値を組み合わせ `domain.SessionFactoryPort` を生成。
      - OpenAI analyzer factory が agent リポジトリ・トークン Vault・レートリミット Provider・openai.Client を用いてアナライザを構築。
      - Result saver adapter が emailstore.UseCase 経由で結果を保存し、枝番の重複をスキップ。
      - Logging adapter が internal/common/application.EmailAnalysisLogUseCase を呼んでパイプラインの進捗を記録。
    - HTTP → 非同期分析の流れ:
      - `internal/emailanalysis/application` が `JobParams` を組み立て、Session/Analyzer Factory で事前チェック後に `PipelineService.Execute` を `go` ルーチンで実行し、Controller は即座にレスポンスを返す。
      - goroutine 内では `runPipeline` が `recover` を仕込んでおり、panic が発生してもプロセス全体は落ちず、メール分析ログおよび構造化ログへ致命情報を残す。
      - エラー再実行方針:
        - パイプラインは fire-and-forget であり、自動リトライやジョブキューは現状存在しない。外部依存（Gmail/OpenAI）エラーで失敗した場合はメール分析ログに失敗理由が記録されるため、ユーザーからの再リクエスト、もしくは Ops による再実行で対応する。
        - OpenAI/Gmail への連続呼び出しは `internal/library/ratelimit.Provider` から注入されたリミッターでガードされており、HTTP リクエストが頻発しても API 側へ過負荷にならない。
        - 追加リトライが必要な場合は messaging ドメインにバックオフ付き再試行を実装する方針とし、成功/失敗の監視は `internal/common/application.EmailAnalysisLogUseCase` 経由のログを CloudWatch 等で可視化する。
      - バックプレッシャ/同時実行数:
        - 1 HTTP リクエスト = 1 goroutine であり、Go ランタイム以外のジョブキューは介在しない。OpenAI/Gmail はリミッターで速度制御し、DB/メール保存は emailstore.UseCase の処理能力に依存する。
        - 大量の同時起動を抑制したい場合は、将来的に HTTP レイヤでキューイングやセマフォ制限を設ける。暫定的にはレートリミット Provider の値（REDIS 経由）を調整し、過度な並列実行を避ける。
      - 監視:
        - メール分析ログにキュー投入／実行／成功／失敗が記録されるため、Ops は最新ログを監視して異常を検知する。
        - アプリ全体の logger も component フィールド込みでエラーを出すので、ログ基盤にアラートを設定する。
  </messaging>

  <context_management>
    - HTTP リクエスト由来の `gin.Context` だけに依存すると、レスポンス返却後のバックグラウンド処理へキャンセルが伝播してしまう。そのため `internal/emailanalysis/application` では `context.WithoutCancel` を利用している。
    - 理想的なフロー:
      - `internal/app/server.Run` で `ctx, cancel := context.WithCancel(context.Background())` を生成し、DI 経由で「サーバ全体のベースコンテキスト」を注入する。
      - バックグラウンド処理は `(baseCtx)` を親に、`context.WithTimeout`/`context.WithCancel` で個別のジョブコンテキストを作り、HTTP リクエスト終了の影響を受けずに実行する。
      - シャットダウン時は `cancel()` → `errgroup` 等で goroutine 完了を待機し、その間にメール分析ログへキャンセルステータスを記録する。
    - 現状の暫定対応:
      - HTTP ctx を `context.WithoutCancel` で分離しつつ、DB/外部接続は各 usecase/adapter が `context.Context` を渡しているため、キャンセル済み接続を再利用することはない。
      - 将来の改善として、`internal/app/server` へベースコンテキストと `errgroup.Group` を導入し、DI 層からメール分析などのバックグラウンドジョブ管理を注入する計画である。
  </context_management>

  <library_layer>
    - 目的: 外部 SDK やクロスカッティングな処理を `internal/library` に閉じ込め、上位レイヤがベンダー SDK へ直接依存しないようにする。
    - 主なパッケージ:
      - `logger`: zap ベースの構造化ログ。`Interface` と `NewNop` を提供。
      - `mysql`: 接続生成・`Transactional` ヘルパー・テスト用 DB 作成。
      - `gmail` / `gmailService`: Gmail API ラッパーと OAuth2 Service 生成。
      - `openai`: レートリミット + リトライ付きのチャットクライアント。
      - `crypto`: HKDF ベースの Vault（agent / emailcredential で利用）。
      - `oswrapper`: 環境変数取得とファイル読み込みの抽象化。
      - `ratelimit`: Redis バックエンドのリミッター Provider（Gmail/OpenAI 用）。
      - `redis`, `retry`, `sendMailerClient`, `timewrapper` などのユーティリティ。
    - 方針:
      - これらのライブラリパッケージはインフラ境界そのものであるため `os.Getenv` を直接利用してよい。アプリケーション/プレゼンテーション層は必ずラッパーの interface を DI で受け取る。
  </library_layer>

  <dependency_injection>
    - 依存注入はすべて `internal/di` に集約する。
      - `di/dig.go` で `ProvideCommonDependencies`（mysql, gmailService, gmail client, OpenAI client, oswrapper, ratelimit Provider, logger, timewrapper, 名前付き limiter）と `BuildContainer` を定義し、機能別モジュールを束ねる。
      - `di/auth.go`, `di/agent.go`, `di/emailstore.go`, `di/email_credential.go`, `di/messaging.go`, `di/presentation.go` 等で各レイヤのコンストラクタを Provide。
      - Messaging モジュールでは emailcredential リポジトリや crypto.Vault、Gmail/OpenAI クライアント、emailstore UseCase、logging UseCase を組み合わせ、emailanalysis UseCase が抽象ポートだけに依存するようにしている。
    - ブートストラップ手順:
      - `cmd/app/main.go` → `internal/app/server.Run` を呼び出す。
      - `server.Run` が oswrapper・logger・MySQL・レートリミット Provider・Gmail/OpenAI クライアントを初期化し、dig コンテナを構築。
      - `internal/app/router.NewRouter` が Gin Engine とコンテナを受け取り、`container.Invoke` で各 Controller/Middleware を 1 度だけ解決した後にルートへ登録。
    - Controller / Middleware は dig を直接意識せず、コンストラクタ注入された依存だけで完結させる。
  </dependency_injection>
</architecture_rules>
