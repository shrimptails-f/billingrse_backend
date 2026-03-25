# 手動メール取得アーキテクチャ 詳細設計

## 本書の位置づけ

- 要件定義は [requirementsDefinition.md](./requirementsDefinition.md) を参照する。
- 基本設計は [basicDesign.md](./basicDesign.md) を参照する。
- 本書は API 契約、永続化スキーマ、application port、stage 集約規則を具体化する。

## 1. HTTP 境界

### 1.1 開始 API

- endpoint
  - `POST /api/v1/manual-mail-workflows`
- 役割
  - 入力検証
  - workflow 受付
  - queued 履歴保存
  - background dispatch
  - `202 Accepted` 返却

request:

```json
{
  "connection_id": 12,
  "label_name": "billing",
  "since": "2026-03-24T00:00:00Z",
  "until": "2026-03-25T00:00:00Z"
}
```

response:

```json
{
  "message": "メール取得ワークフローを受け付けました。",
  "workflow_id": "01JQ0B7N0M7H3X9C2J5K8V6P4",
  "status": "queued"
}
```

### 1.2 状態取得 API

- endpoint
  - `GET /api/v1/manual-mail-workflows/:workflow_id`
- 役割
  - workflow の現在状態を返す
  - 完了済みなら stage ごとの件数と failure 理由を返す
  - `manual_mail_workflow_histories` と `manual_mail_workflow_stage_failures` を読み出して DTO を組み立てる

response:

```json
{
  "workflow_id": "01JQ0B7N0M7H3X9C2J5K8V6P4",
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
        "reason_code": "provider_fetch_failed",
        "message": "メールの取得に失敗しました。"
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
        "message": "支払先を特定できませんでした。"
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
        "reason_code": "missing_billing_number",
        "message": "請求番号が不足しているため請求を作成できませんでした。"
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
        "message": "同じ請求番号の請求が既に存在します。"
      }
    ]
  }
}
```

### 1.3 状態値と stage 値

| 項目 | 値 |
| --- | --- |
| `status` | `queued`, `running`, `succeeded`, `partial_success`, `failed` |
| `current_stage` | `fetch`, `analysis`, `vendorresolution`, `billingeligibility`, `billing` |

## 2. application 設計

### 2.1 workflow 受付と background 実行の分離

現行実装では `application.Command` が開始入力と実行入力を兼ねているが、履歴保存を追加するにあたり責務を明確に分離する。

```go
type StartCommand struct {
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

type StartResult struct {
	WorkflowID string
	Status     string
}

type DispatchJob struct {
	HistoryID    uint64
	WorkflowID   string
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}
```

### 2.2 StartUseCase

```go
type StartUseCase interface {
	Start(ctx context.Context, cmd StartCommand) (StartResult, error)
}
```

責務:

1. `ctx`、`user_id`、`connection_id`、`FetchCondition` を検証する。
2. `workflow_id` を採番する。
3. `queued` 状態の履歴 header row を作成し、`history_id` を受け取る。
4. `history_id` と `workflow_id` を含む job を dispatcher に渡す。
5. dispatch 失敗時は履歴を `failed` に更新する。
6. `workflow_id` と `queued` 状態を返す。

### 2.3 Runner

workflow 実行本体は `manualmailworkflow` の runner が担う。既存の `UseCase.Execute` を runner として延長するか、新しい `RunnerUseCase` を導入してもよいが、責務は以下で固定する。

```go
type RunnerUseCase interface {
	Run(ctx context.Context, job DispatchJob) error
}
```

責務:

1. workflow を `running` に遷移させる。
2. `current_stage` を更新しながら stage を順に呼ぶ。
3. 各 stage の success / failure count と failure row を保存する。
4. skip 条件を満たした stage は実行せず、次の状態判定へ進む。
5. 全 stage 終了後に `succeeded` / `partial_success` / `failed` を確定する。

### 2.4 状態取得 usecase

```go
type GetStatusQuery struct {
	UserID     uint
	WorkflowID string
}

type StageFailureMessage struct {
	ExternalMessageID *string
	ReasonCode        string
	Message           string
}

type StageSummary struct {
	SuccessCount int
	FailureCount int
	Failures     []StageFailureMessage
}

type GetStatusResult struct {
	WorkflowID         string
	Status             string
	CurrentStage       *string
	QueuedAt           time.Time
	FinishedAt         *time.Time
	Fetch              StageSummary
	Analysis           StageSummary
	VendorResolution   StageSummary
	BillingEligibility StageSummary
	Billing            StageSummary
}
```

### 2.5 stage ports

```go
type FetchStage interface {
	Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

type AnalyzeStage interface {
	Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

type VendorResolutionStage interface {
	Execute(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error)
}

type BillingEligibilityStage interface {
	Execute(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error)
}

type BillingStage interface {
	Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}
```

## 3. 永続化設計

### 3.1 repository port

```go
type QueuedWorkflowHistory struct {
	WorkflowID   string
	UserID       uint
	ConnectionID uint
	LabelName    string
	SinceAt      time.Time
	UntilAt      time.Time
	QueuedAt     time.Time
}

type WorkflowHistoryRef struct {
	HistoryID  uint64
	WorkflowID string
}

type StageFailureRecord struct {
	Stage             string
	ExternalMessageID *string
	ReasonCode        string
	Message           string
}

type StageProgress struct {
	HistoryID      uint64
	Stage          string
	SuccessCount   int
	FailureCount   int
	FailureRecords []StageFailureRecord
}

type WorkflowStatusRepository interface {
	CreateQueued(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error)
	MarkRunning(ctx context.Context, historyID uint64, currentStage string) error
	SaveStageProgress(ctx context.Context, progress StageProgress) error
	Complete(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error
	Fail(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time) error
	FindByWorkflowID(ctx context.Context, userID uint, workflowID string) (GetStatusResult, error)
}
```

方針:

- `SaveStageProgress` は header table の count 更新と failure row insert を同一 transaction で行う。
- failure row は stage から返された明細をそのまま insert し、workflow 層では dedupe しない。
- `FailureCount` と `FailureRecords` の整合は各 stage が保証する。
- `Fail` は途中までの count / failure rows を残したまま `failed` へ遷移させる。
- `FindByWorkflowID` は header と failure rows から API 向け DTO を再構築する。

### 3.2 `manual_mail_workflow_histories`

```sql
CREATE TABLE `manual_mail_workflow_histories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `workflow_id` char(26) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `provider` varchar(50) NOT NULL,
  `account_identifier` varchar(255) NOT NULL,
  `label_name` varchar(255) NOT NULL,
  `since_at` datetime(3) NOT NULL,
  `until_at` datetime(3) NOT NULL,
  `status` varchar(32) NOT NULL,
  `current_stage` varchar(32) NULL,
  `queued_at` datetime(3) NOT NULL,
  `finished_at` datetime(3) NULL,
  `fetch_success_count` int NOT NULL DEFAULT 0,
  `fetch_business_failure_count` int NOT NULL DEFAULT 0,
  `fetch_technical_failure_count` int NOT NULL DEFAULT 0,
  `analysis_success_count` int NOT NULL DEFAULT 0,
  `analysis_business_failure_count` int NOT NULL DEFAULT 0,
  `analysis_technical_failure_count` int NOT NULL DEFAULT 0,
  `vendor_resolution_success_count` int NOT NULL DEFAULT 0,
  `vendor_resolution_business_failure_count` int NOT NULL DEFAULT 0,
  `vendor_resolution_technical_failure_count` int NOT NULL DEFAULT 0,
  `billing_eligibility_success_count` int NOT NULL DEFAULT 0,
  `billing_eligibility_business_failure_count` int NOT NULL DEFAULT 0,
  `billing_eligibility_technical_failure_count` int NOT NULL DEFAULT 0,
  `billing_success_count` int NOT NULL DEFAULT 0,
  `billing_business_failure_count` int NOT NULL DEFAULT 0,
  `billing_technical_failure_count` int NOT NULL DEFAULT 0,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uni_manual_mail_workflow_histories_workflow_id` (`workflow_id`),
  INDEX `idx_manual_mail_workflow_histories_user_queued_at` (`user_id`, `queued_at`),
  INDEX `idx_manual_mail_workflow_histories_user_status_queued_at` (`user_id`, `status`, `queued_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

補足:

- `workflow_id` は API 参照用の一意キーとする。
- `provider` と `account_identifier` は workflow 受付時点のメール連携 snapshot を保持する。
- `queued_at` は保持するが、`started_at` は持たない。
- stage summary は一覧 API でも再利用できるよう header 側に持つ。

### 3.3 `manual_mail_workflow_stage_failures`

```sql
CREATE TABLE `manual_mail_workflow_stage_failures` (
  `workflow_history_id` bigint unsigned NOT NULL,
  `stage` varchar(32) NOT NULL,
  `external_message_id` varchar(255) NULL,
  `reason_code` varchar(64) NOT NULL,
  `message` varchar(255) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  INDEX `idx_manual_mail_workflow_stage_failures_history_stage_created_at`
    (`workflow_history_id`, `stage`, `created_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

補足:

- `id` は持たない。`ManualMailWorkflowHistory` 配下の単純な明細として append-only で保持する。
- `external_message_id` は message 単位へ落とせる failure で使う。
- stage 全体 failure のように message 単位へ分解できない場合は `NULL` を許容する。
- `message` は各 stage の `Execute` が返す明細文言を保存する。多言語対応は行わない。
- stage が返した failure 明細をそのまま保存するため、header の `business_failure_count + technical_failure_count` と failure rows の件数は一致する前提とする。

## 4. 件数定義

- `fetch_success_count`
  - `created_email_count + existing_email_count`
- `fetch_business_failure_count`
  - `0`
- `fetch_technical_failure_count`
  - `len(fetch.Failures)`
- `analysis_success_count`
  - `parsed_email_count`
- `analysis_business_failure_count`
  - `0`
- `analysis_technical_failure_count`
  - `len(analysis.Failures)`
- `vendor_resolution_success_count`
  - `resolved_count`
- `vendor_resolution_business_failure_count`
  - `unresolved_count`
- `vendor_resolution_technical_failure_count`
  - `len(vendorresolution.Failures)`
- `billing_eligibility_success_count`
  - `eligible_count`
- `billing_eligibility_business_failure_count`
  - `ineligible_count`
- `billing_eligibility_technical_failure_count`
  - `len(billingeligibility.Failures)`
- `billing_success_count`
  - `created_count`
- `billing_business_failure_count`
  - `duplicate_count`
- `billing_technical_failure_count`
  - `len(billing.Failures)`

## 5. failure row への写像

- `fetch`
  - `FetchFailure` を `stage=fetch` として保存し、stage が返した `message` もそのまま保存する。
- `analysis`
  - `AnalysisFailure` を `stage=analysis` として保存し、stage が返した `message` もそのまま保存する。
- `vendorresolution`
  - unresolved 明細は `reason_code=vendor_unresolved` と `message` を stage が返す。
  - technical failure も `VendorResolutionFailure.Code` と `message` を stage が返す。
- `billingeligibility`
  - `IneligibleItem.ReasonCode` と `message` を `stage=billingeligibility` の failure row として保存する。
  - technical failure も `BillingEligibilityFailure.Code` と `message` を stage が返す。
- `billing`
  - `DuplicateItem` は `reason_code=duplicate_billing` と `message` を stage が返す。
  - technical failure も `BillingFailure.Code` と `message` を stage が返す。

## 6. runner 制御

### 6.1 実行順序

1. `MarkRunning(historyID, "fetch")`
2. `mailfetch` 実行
3. fetch の `SaveStageProgress`
4. `current_stage=analysis` を保存して `mailanalysis` 実行
5. analysis の `SaveStageProgress`
6. `current_stage=vendorresolution` を保存して `vendorresolution` 実行
7. vendorresolution の `SaveStageProgress`
8. `current_stage=billingeligibility` を保存して `billingeligibility` 実行
9. billingeligibility の `SaveStageProgress`
10. `current_stage=billing` を保存して `billing` 実行
11. billing の `SaveStageProgress`
12. 全 stage の count を集約して最終 status を決定
13. `Complete(historyID, status, finishedAt)` を呼ぶ

### 6.2 skip 条件

- `analysis.ParsedEmails == 0`
  - `vendorresolution`、`billingeligibility`、`billing` は実行しない。
- `vendorresolution.ResolvedItems == 0`
  - `billingeligibility`、`billing` は実行しない。
- `billingeligibility.EligibleItems == 0`
  - `billing` は実行しない。

### 6.3 status 判定

- `failed`
  - dispatch 失敗、stage top-level error、panic などで workflow が完走しなかった場合
- `partial_success`
  - workflow は完走したが、いずれかの stage の `*_business_failure_count` または `*_technical_failure_count` が 1 件以上ある場合
- `succeeded`
  - workflow が完走し、かつどの stage にも failure が残っていない場合

### 6.4 panic / top-level error

- panic は recover してログへ出すだけで終わらせず、履歴も `failed` に更新する。
- top-level error が返った stage では、それ以前に保存済みの stage progress は残す。
- `Fail` では `finished_at` を保存し、`current_stage` は失敗した stage 名を残す。

## 7. dispatcher / adapter 設計

### 7.1 dispatcher

```go
type WorkflowDispatcher interface {
	Dispatch(ctx context.Context, job DispatchJob) error
}
```

初期方針:

- まずは `InProcessWorkflowDispatcher` を使う。
- background 実行では新しい `context.Context` を作り、`request_id`、`job_id`、`user_id` を引き継ぐ。
- queue 製品に切り替える場合も application は `WorkflowDispatcher` だけを見る。

### 7.2 adapter 一覧

- `DirectManualMailFetchAdapter`
  - `mailfetch.UseCase` を呼ぶ
- `DirectMailAnalysisAdapter`
  - `mailanalysis.UseCase` を呼ぶ
- `DirectVendorResolutionAdapter`
  - `vendorresolution.UseCase` を呼ぶ
- `DirectBillingEligibilityAdapter`
  - `billingeligibility.UseCase` を呼ぶ
- `DirectBillingAdapter`
  - `billing.UseCase` を呼ぶ
- `WorkflowStatusRepositoryAdapter`
  - workflow 履歴 header と stage failure table の read / write を担う

## 8. DI / migration

- `internal/di/manualmailworkflow.go`
  - start usecase
  - runner
  - get status usecase
  - dispatcher
  - workflow status repository
  - controller
  を組み立てる。
- Atlas migration で以下を追加する。
  - `manual_mail_workflow_histories`
  - `manual_mail_workflow_stage_failures`

## 9. テスト観点

- `StartUseCase`
  - 入力不正
  - queued 保存成功
  - dispatch 失敗時の `failed` 更新
- `Runner`
  - stage 順実行
  - skip 条件
  - partial_success 判定
  - panic / top-level error 時の `failed` 更新
- `WorkflowStatusRepositoryAdapter`
  - `CreateQueued`
  - `SaveStageProgress` の transaction 性
  - failure 明細を dedupe せず保存できること
  - `FindByWorkflowID` の DTO 再構築
- `Controller`
  - `202 Accepted`
  - 将来の `GET` 契約
  - failure `message` が安全な文言で返ること
