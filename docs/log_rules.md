# ログ設計ルール

## 目的

本ドキュメントは、アプリケーションログの設計・実装・運用ルールを定義する。

- 障害対応: 問題発生時に原因調査と影響範囲の特定をしやすくする
- 性能分析: 遅延やタイムアウトの発生箇所を把握しやすくする
- 監査補助: 重要な処理の実行有無を追跡できるようにする

## 対象範囲

- HTTP API のリクエスト処理
- 認証 / 認可に関わる重要イベント
- 外部 API 呼び出し
- DB エラーと slow query
- panic recovery

バッチ / 非同期処理のログも対象に含めるが、現時点では `job_id` の context helper までが実装済みで、ジョブイベントの本格運用は未着手である。

## 基本方針

- ログは構造化 JSON で `stdout` に出力する
- 人が読む用途と、集計・検索する用途の両方を考慮する
- HTTP リクエストではサマリログを最低 1 件残す
- フィールド名は固定化し、互換性を意識して追加・変更する
- request-scoped な項目は `context.Context` 経由で lower layer まで伝播する
- `nil context` は暗黙に吸収せず、原則 `error` として扱う

## 現在の実装ルール

### HTTP リクエスト

- `http_request_started`
  - リクエスト受付時の開始ログ
- `http_request_succeeded`
  - 2xx / 3xx のサマリログ
- `http_request_rejected`
  - 想定内の 4xx サマリログ
- `http_request_failed`
  - 5xx サマリログ

HTTP サマリでは次の追加項目を出力する。

- `method`
- `path`
- `http_status_code`
- `latency_ms`

### 認証 / 認可

- `login_succeeded`
  - ログイン成功時
- `login_failed`
  - ログイン失敗時
- `permission_denied`
  - 認証 / 認可で処理を拒否したとき

### 外部依存

- `external_api_succeeded`
  - OpenAI / Gmail / SMTP などの外部依存呼び出し成功時
- `external_api_failed`
  - 外部依存呼び出し失敗時
- `db_query_failed`
  - DB エラー発生時
- `db_query_slow`
  - slow query 検知時

### panic recovery

- panic recovery では `message: "panic recovered"` を出力する
- `stack_trace` を構造化フィールドとして含める

## 共通スキーマ

全ログで次のフィールドを基本とする。

| フィールド | 必須 | 説明 |
| --- | --- | --- |
| `level` | 必須 | `debug` / `info` / `warn` / `error` |
| `service` | 必須 | サービス名。root logger の固定項目 |
| `environment` | 必須 | 実行環境名。root logger の固定項目 |
| `component` | 必須 | controller / middleware / repository / client などの責務単位 |
| `message` | 必須 | 人間向けの短い説明。イベント名は snake_case を基本とする |
| `request_id` | HTTP では必須 | HTTP 相関 ID。HTTP 以外では未設定時に省略する |
| `job_id` | バッチ / 非同期では必須 | ジョブ相関 ID。未設定時に省略する |
| `user_id` | 任意 | 認証済みユーザー ID。未設定時に省略する |
| `http_status_code` | 任意 | HTTP ステータスコード |
| `stack_trace` | 任意 | panic recovery などで必要なときのみ出力する |

文脈に応じて次の追加項目を使う。

- HTTP サマリ: `method`, `path`, `latency_ms`
- 外部 API: `provider`, `operation`
- DB: `db_system`, `db_operation`, `table`, `operation`, `rows_affected`, `latency_ms`

## ログレベル方針

- `debug`
  - ローカル開発や一時調査用
- `info`
  - 正常系の主要イベント
  - 想定内の 4xx
  - `login_failed`, `permission_denied`
- `warn`
  - slow query
  - 調査対象だが即時障害ではない異常
- `error`
  - 5xx
  - `external_api_failed`
  - `db_query_failed`
  - panic recovery

## 相関ルール

- HTTP リクエストでは middleware で `request_id` を生成または受け取り、`gin.Context` と `context.Context` の両方に積む
- 認証済みリクエストでは `user_id` を `context.Context` に積み、controller だけでなく repository / client でも参照できるようにする
- バッチ / 非同期処理では `job_id` を `context.Context` に積む
- logger は `WithContext(ctx)` で `request_id` / `job_id` / `user_id` を自動付与する

## 秘匿情報 / PII ルール

次の情報はログへそのまま出力しない。

- メールアドレス
- 電話番号
- 住所
- パスワード
- トークン
- API キー
- Cookie
- Authorization ヘッダ
- セッション ID
- 個人情報を含むリクエストボディ全文
- 外部 API の生ペイロード全文

必要な場合は次のどちらかで扱う。

- マスクする
- 件数・有無・サイズなど、識別できない形に要約する

## 出力先 / 保持

- アプリケーションは JSON を `stdout` に出力する
- 保管・検索は CloudWatch 集約を前提とする
- 保持期間、S3 退避、アラート、メトリクス設計はインフラ / 運用設計で別途定義する
