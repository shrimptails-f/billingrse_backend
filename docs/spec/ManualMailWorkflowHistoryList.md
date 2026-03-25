# 手動メール取得履歴一覧 API 仕様

本ドキュメントは、手動メール取得履歴一覧 API の最新の要件定義・設計内容を 1 ファイルに集約した仕様書である。

参照元:
- `docs/architecture.md`
- `docs/api_design.md`
- `docs/spec/manualmailworkflow/requirementsDefinition.md`
- `docs/spec/manualmailworkflow/basicDesign.md`
- `docs/spec/manualmailworkflow/detailDesign.md`

## 1. 概要

### 背景
- 既存実装では `POST /api/v1/manual-mail-workflows` により workflow を受け付け、`manual_mail_workflow_histories` と `manual_mail_workflow_stage_failures` に履歴を保存する。
- `manualmailworkflow` 配下の設計では単体状態取得 API が想定されているが、履歴を複数件一覧する API は未提供である。
- frontend では、workflow ごとの受付条件・進行状態・stage 件数・failure 明細を一覧画面で確認したい。

### 目的
- 認証済みユーザーが、自分自身の手動メール取得履歴を `GET` で一覧取得できるようにする。
- 1 件ごとに header row と child row の両方を返し、failure 明細も一覧画面で確認できるようにする。
- `limit` / `offset` ベースのページネーションを採用する。
- child row 読み出しで N+1 を起こさず、固定本数のクエリで取得できるようにする。

### 非スコープ
- cursor pagination
- 並び順の動的切り替え
- 複雑な検索条件の追加
- 他ユーザーの履歴参照
- バッチメール取得履歴

## 2. API 契約

### Endpoint
- Method: `GET`
- Path: `/api/v1/manual-mail-workflows`
- Auth: required

### Query
- `limit`
  - 任意
  - 1 以上 100 以下
  - 未指定時は `20`
- `offset`
  - 任意
  - 0 以上
  - 未指定時は `0`
- `status`
  - 任意
  - `queued`, `running`, `succeeded`, `partial_success`, `failed`

### Response 200

```json
{
  "items": [
    {
      "workflow_id": "01JQ0B7N0M7H3X9C2J5K8V6P4",
      "connection_id": 12,
      "label_name": "billing",
      "since": "2026-03-24T00:00:00Z",
      "until": "2026-03-25T00:00:00Z",
      "status": "partial_success",
      "current_stage": null,
      "queued_at": "2026-03-25T17:00:00Z",
      "finished_at": "2026-03-25T17:00:12Z",
      "fetch": {
        "success_count": 14,
        "business_failure_count": 0,
        "technical_failure_count": 1,
        "failures": [
          {
            "external_message_id": "18c1f3...",
            "reason_code": "fetch_detail_failed",
            "message": "メールの取得に失敗しました。",
            "created_at": "2026-03-25T17:00:02Z"
          }
        ]
      },
      "analysis": {
        "success_count": 14,
        "business_failure_count": 0,
        "technical_failure_count": 0,
        "failures": []
      },
      "vendor_resolution": {
        "success_count": 12,
        "business_failure_count": 2,
        "technical_failure_count": 0,
        "failures": [
          {
            "external_message_id": "18c1fa...",
            "reason_code": "vendor_unresolved",
            "message": "支払先を特定できませんでした。",
            "created_at": "2026-03-25T17:00:09Z"
          }
        ]
      },
      "billing_eligibility": {
        "success_count": 10,
        "business_failure_count": 2,
        "technical_failure_count": 0,
        "failures": [
          {
            "external_message_id": "18c1fb...",
            "reason_code": "billing_number_empty",
            "message": "請求番号が不足しているため請求を作成できませんでした。",
            "created_at": "2026-03-25T17:00:10Z"
          }
        ]
      },
      "billing": {
        "success_count": 8,
        "business_failure_count": 2,
        "technical_failure_count": 0,
        "failures": [
          {
            "external_message_id": "18c1fc...",
            "reason_code": "duplicate_billing",
            "message": "同じ請求番号の請求が既に存在します。",
            "created_at": "2026-03-25T17:00:11Z"
          }
        ]
      }
    }
  ],
  "total_count": 57
}
```

### Response field
- `items`
  - 現在ページに含まれる workflow 履歴一覧
- `total_count`
  - filter 適用後の総件数
- `workflow_id`
  - API 参照用の一意な workflow 識別子
- `connection_id`
  - 実行対象のメール連携 ID
- `label_name`
  - 受付時点のラベル条件
- `since`
  - 受付時点の取得期間開始
- `until`
  - 受付時点の取得期間終了
- `status`
  - `queued`, `running`, `succeeded`, `partial_success`, `failed`
- `current_stage`
  - `running` 中のみ stage 値を持つ
  - それ以外は `null`
- `queued_at`
  - workflow 受付時刻
- `finished_at`
  - terminal status になった時刻。未完了時は `null`
- `fetch`, `analysis`, `vendor_resolution`, `billing_eligibility`, `billing`
  - stage ごとの件数と failure 明細
- `failures`
  - `manual_mail_workflow_stage_failures` の child row を stage ごとに束ねて返す
- `failures[].external_message_id`
  - child row の `external_message_id`
  - stage 全体 failure など message 単位に落ちない場合は `null`
- `failures[].reason_code`
  - child row の `reason_code`
- `failures[].message`
  - child row の `message`
- `failures[].created_at`
  - child row の `created_at`

### Error
- `400 invalid_request`
  - `limit` が範囲外
  - `offset` が負数
  - `status` が不正
- `401 unauthorized`
  - JWT 不正または未認証
- `500 internal_server_error`
  - DB 読み出し失敗など、一覧取得に失敗した場合

### 契約上の注意
- JSON フィールド名は `lower_snake_case` とする。
- コレクションレスポンスは `items` を基本キーとする。
- stage summary の shape は将来の単体取得 API と揃える。
- v1 は `limit` / `offset` を採用するため、ページング中に新しい履歴が追加されるとページずれは起こり得る。

## 3. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身が所有する履歴のみ取得できること。
- header row の内容と child row の failure 明細を同一レスポンスで返せること。
- `status` による絞り込みができること。
- stage 件数は `manual_mail_workflow_histories` の集計カラムをそのまま返すこと。
- failure 明細は `manual_mail_workflow_stage_failures` の保存内容を漏れなく返すこと。
- 1 件の workflow に failure row が存在しない stage では、`failures` は空配列を返すこと。
- `queued_at` の降順を基本並び順とし、同時刻タイブレークには `id` 降順を使うこと。
- レスポンスや構造化ログにメール本文や OAuth token を出さないこと。

## 4. 取得方針

### ページング
- v1 は `limit` / `offset` を採用する。
- `limit` は default `20`、max `100` とする。
- `offset` は default `0` とする。
- response には `total_count` を含める。

### 並び順
- `ORDER BY queued_at DESC, id DESC`
- `queued_at` のみでは同値があり得るため、`id` を tie-breaker に使う。

### 取得アルゴリズム
1. query parameter を検証し、`limit` と `offset` を正規化する。
2. `user_id` と任意の `status` 条件で `manual_mail_workflow_histories` の `COUNT(*)` を取得する。
3. 同じ filter 条件で header row を 1 ページ分取得する。
4. header row が 0 件なら `items: []` を返し、child table は読みに行かない。
5. header row から `history_id` 一覧を集める。
6. `workflow_history_id IN (...)` で `manual_mail_workflow_stage_failures` を一括取得する。
7. application 側で `workflow_history_id` と `stage` ごとに group 化し、各 item の stage summary に詰める。
8. `items` と `total_count` を返す。

## 5. N+1 回避とクエリ設計

### 基本方針
- 1 workflow ごとに child table を読む実装は禁止する。
- 1 request あたりの DB 読み出しは最大 3 クエリに固定する。
  - `COUNT(*)`
  - header page 取得
  - child failure 一括取得
- page が空の場合は child failure query を省略し、最大 2 クエリとする。

### 想定 SQL

header 件数取得:

```sql
SELECT COUNT(*)
FROM manual_mail_workflow_histories
WHERE user_id = :user_id
  AND (:status IS NULL OR status = :status);
```

header page 取得:

```sql
SELECT
  id,
  workflow_id,
  connection_id,
  label_name,
  since_at,
  until_at,
  status,
  current_stage,
  queued_at,
  finished_at,
  fetch_success_count,
  fetch_business_failure_count,
  fetch_technical_failure_count,
  analysis_success_count,
  analysis_business_failure_count,
  analysis_technical_failure_count,
  vendor_resolution_success_count,
  vendor_resolution_business_failure_count,
  vendor_resolution_technical_failure_count,
  billing_eligibility_success_count,
  billing_eligibility_business_failure_count,
  billing_eligibility_technical_failure_count,
  billing_success_count,
  billing_business_failure_count,
  billing_technical_failure_count
FROM manual_mail_workflow_histories
WHERE user_id = :user_id
  AND (:status IS NULL OR status = :status)
ORDER BY queued_at DESC, id DESC
LIMIT :limit OFFSET :offset;
```

child failure 一括取得:

```sql
SELECT
  workflow_history_id,
  stage,
  external_message_id,
  reason_code,
  message,
  created_at
FROM manual_mail_workflow_stage_failures
WHERE workflow_history_id IN (:history_ids)
ORDER BY workflow_history_id ASC, stage ASC, created_at ASC;
```

### index 方針
- 既存の `idx_manual_mail_workflow_histories_user_queued_at`
  - `user_id`, `queued_at`
- 既存の `idx_manual_mail_workflow_histories_user_status_queued_at`
  - `user_id`, `status`, `queued_at`
- 一覧 API の tie-breaker と offset scan を安定させるため、次の index 追加を推奨する。
  - `idx_manual_mail_workflow_histories_user_queued_at_id (user_id, queued_at, id)`
  - `idx_manual_mail_workflow_histories_user_status_queued_at_id (user_id, status, queued_at, id)`
- child table は既存の `idx_manual_mail_workflow_stage_failures_history_stage_created_at` を利用する。

### 実装上の注意
- Gorm の `Preload` を安易に使って item ごとに child を引く構成にはしない。
- child query は page header の `history_id` 群に対する 1 回の `IN` query で行う。
- group 化は application もしくは infrastructure 内の assembler で行い、controller では行わない。

## 6. レイヤ設計

### Presentation
- `GET /api/v1/manual-mail-workflows` を追加する。
- handler は `(*Controller).List` とする。
- query parameter は controller で parse し、application query DTO に変換する。
- `limit`, `offset`, `status` の構文不正は `400 invalid_request` を返す。

### Application
- `manualmailworkflow` に一覧取得 usecase を追加する。

```go
type ListQuery struct {
    UserID uint
    Limit  int
    Offset int
    Status *string
}

type StageFailureView struct {
    ExternalMessageID *string
    ReasonCode        string
    Message           string
    CreatedAt         time.Time
}

type StageSummaryView struct {
    SuccessCount          int
    BusinessFailureCount  int
    TechnicalFailureCount int
    Failures              []StageFailureView
}

type WorkflowHistoryListItem struct {
    WorkflowID         string
    ConnectionID       uint
    LabelName          string
    Since              time.Time
    Until              time.Time
    Status             string
    CurrentStage       *string
    QueuedAt           time.Time
    FinishedAt         *time.Time
    Fetch              StageSummaryView
    Analysis           StageSummaryView
    VendorResolution   StageSummaryView
    BillingEligibility StageSummaryView
    Billing            StageSummaryView
}

type ListResult struct {
    Items      []WorkflowHistoryListItem
    TotalCount int64
}
```

- 一覧取得 usecase は query 検証と結果返却に責務を限定する。
- workflow 実行 usecase や start usecase とは分離する。

### Infrastructure
- read 専用 repository port を追加する。

```go
type WorkflowHistoryListRepository interface {
    List(ctx context.Context, query ListQuery) (ListResult, error)
}
```

- 実装は `manual_mail_workflow_histories` と `manual_mail_workflow_stage_failures` を固定本数クエリで読み出す。
- `manual_mail_workflow_stage_failures` の group 化は repository 実装内で完結してよい。

### Router / DI
- router に `GET /api/v1/manual-mail-workflows` を追加する。
- 既存 `manualmailworkflow` controller に `List` handler を追加する。
- DI では start usecase とは別に list usecase を controller へ注入する。

## 7. テスト観点

### Controller
- `200 OK` 正常系
- `400 invalid_request`
  - `limit=0`
  - `limit=101`
  - `offset=-1`
  - 不正な `status`
- `401 unauthorized`
- `500 internal_server_error`

### UseCase
- default `limit=20`, `offset=0`
- `status=nil` と `status=partial_success`
- `total_count` を保ったまま items を返せること
- query 検証失敗時に `invalid_request` 相当の error を返せること

### Repository
- `user_id` で自分の履歴だけ取得できること
- `status` filter が効くこと
- `ORDER BY queued_at DESC, id DESC` で安定順序になること
- `limit` / `offset` が効くこと
- page 内の複数 workflow に対して failure 明細を一括取得できること
- stage ごとの `failures` が正しく group 化されること
- page が空のとき child query を発行しないこと

### Query efficiency
- 一覧 API の DB query 数が item 件数に比例して増えないこと
- integration test または SQL mock で、最大 3 query で完結することを確認する

### Router
- `GET /api/v1/manual-mail-workflows` が登録されること

## 8. 判断事項

- v1 のページネーションは `limit` / `offset` を採用する。
- v1 一覧 API は child table の failure 明細も返す。
- v1 一覧 API は stage 件数を header 集計カラムから返し、件数再計算はしない。
- N+1 回避のため、repository は item ごとの child 読み出しを禁止する。
- sort は `queued_at DESC, id DESC` で固定する。
