# 手動メール取得アーキテクチャ 設計

## 1. 設計方針
- `manualmailworkflow` は同期完了型ではなく、非同期 workflow の受付と状態参照を担う package とする。
- `POST /api/v1/manual-mail-workflows` は受付専用とし、workflow の実処理は background runner が担う。
- stage 実行順序は `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` のまま維持する。
- stage 間のデータ受け渡しは、既存実装と同様に workflow payload を優先する。
- workflow 履歴テーブルや最終結果保存テーブルの詳細は今回は深掘りせず、`TODO:` として残す。

## 2. 推奨ディレクトリ

### `internal/manualmailworkflow`
- `application`
  - workflow 開始 usecase
  - background runner usecase
  - workflow 状態取得 usecase
  - 受付結果 / 状態結果 DTO
  - stage 実行結果の集約
- `infrastructure`
  - stage direct adapter
  - async dispatcher
  - workflow 状態 repository adapter
  - 将来必要なら queue adapter

補足:
- `manualmailworkflow` は orchestration 専用のため、基本的に独自 domain は持たない。
- workflow 状態保存の詳細スキーマは `TODO:` とし、application は port だけを先に持つ。

### `internal/mailfetch`
- `application`
  - 手動メール取得の usecase
  - 入力検証
  - 外部メール取得のオーケストレーション
  - `Email` 保存
  - `created_email_ids` / `created_emails` / `existing_email_ids` の返却
- `domain`
  - 手動メール取得に閉じた概念
  - `FetchCondition`
- `infrastructure`
  - Gmail 用 `MailFetcher` adapter
  - mail account connection / credential 解決 adapter
  - `EmailRepository` の Gorm 実装

### `internal/mailanalysis`
- `application`
  - メール解析 usecase
  - prompt 生成のオーケストレーション
  - AI 実行
  - `ParsedEmail` 保存
  - 保存済み `parsed_email_ids` と workflow 向け `parsed_emails` payload の返却
- `domain`
  - `ParsedEmail` に関する概念
  - 解析結果の検証ルール
- `infrastructure`
  - OpenAI analyzer adapter
  - prompt builder
  - `ParsedEmailRepository` の Gorm 実装

### `internal/vendorresolution`
- `application`
  - vendor 解決 usecase
  - workflow payload として受け取った `ParsedEmail` / source email 情報の利用
  - vendor master / alias facts の取得
  - `VendorResolution` 適用
  - resolved / unresolved の返却
- `domain`
  - `VendorResolutionInput`
  - `ParsedEmailForResolution`
  - `SourceEmail`
  - `ResolvedItem`
  - `Failure`
- `infrastructure`
  - vendor master / alias 読み出し adapter
  - 未解決時の canonical Vendor 補完 adapter
  - `VendorResolutionRepository` / `VendorRegistrationRepository` の Gorm 実装

### `internal/billingeligibility`
- `application`
  - 請求成立判定 usecase
  - `VendorResolution` の結果受け取り
  - 請求成立条件の評価
  - eligible / ineligible の返却
- `domain`
  - `EligibleItem`
  - `IneligibleItem`
  - `Failure`
- `infrastructure`
  - 初期実装では不要
  - workflow payload を入力にするため repository は持たない

### `internal/billing`
- `application`
  - `Billing` 生成 usecase
  - `BillingEligibility` の結果受け取り
  - 重複確認を含む idempotent 保存
  - 導出結果サマリ返却
- `domain`
  - stage 専用の result / failure model
  - `Billing`, `Money`, `BillingNumber`, `InvoiceNumber`, `PaymentCycle` は `internal/common/domain` を再利用する
- `infrastructure`
  - `BillingRepository` の Gorm 実装

## 3. HTTP 境界の設計

### 3.1 開始 API
- endpoint:
  - `POST /api/v1/manual-mail-workflows`
- 役割:
  - 入力検証
  - workflow 受付
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
  "workflow_id": "01HXYZ...",
  "status": "queued"
}
```

### 3.2 状態取得 API
- endpoint:
  - `GET /api/v1/manual-mail-workflows/:workflow_id`
- 役割:
  - workflow の現在状態を返す
  - 完了済みなら最終結果も返す

状態値:
- `queued`
- `running`
- `succeeded`
- `partial_success`
- `failed`

補足:
- `partial_success` は workflow 自体は完走したが、stage failure / unresolved / ineligible / duplicate などを含む状態を指す。
- 履歴保存の詳細は `TODO:` だが、API 契約としては `workflow_id` による参照を前提にする。

## 4. `manualmailworkflow` application 設計

### 4.1 `StartUseCase`
役割:
1. `ctx`、`user_id`、`connection_id`、`FetchCondition` を検証する。
2. `workflow_id` を採番する。
3. workflow 受付状態を repository に保存する。
4. dispatcher に background 実行を依頼する。
5. `workflow_id` と `queued` 状態を返す。

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
```

### 4.2 `RunnerUseCase`
役割:
1. workflow を `running` に遷移させる。
2. `mailfetch` を実行する。
3. `mailanalysis` を実行する。
4. `vendorresolution` を実行する。
5. `billingeligibility` を実行する。
6. `billing` を実行する。
7. stage 結果を集約し、`succeeded` / `partial_success` / `failed` を決定する。
8. workflow の最終状態を保存する。

補足:
- stage 実行順序は同期版と同じで、background に移るだけである。
- `analysis.ParsedEmails == 0` なら `vendorresolution` を skip する。
- `vendorresolution.ResolvedItems == 0` なら `billingeligibility` を skip する。
- `billingeligibility.EligibleItems == 0` なら `billing` を skip する。

### 4.3 `GetStatusUseCase`
役割:
1. `workflow_id` を受け取る。
2. workflow の最新状態を repository から取得する。
3. 状態と、取得可能なら stage 結果を返す。

```go
type GetStatusQuery struct {
	UserID      uint
	WorkflowID  string
}

type GetStatusResult struct {
	WorkflowID         string
	Status             string
	Fetch              *FetchResult
	Analysis           *AnalyzeResult
	VendorResolution   *VendorResolutionResult
	BillingEligibility *BillingEligibilityResult
	Billing            *BillingResult
}
```

## 5. workflow 状態 repository

```go
type WorkflowStatusRepository interface {
	CreateQueued(ctx context.Context, snapshot QueuedWorkflowSnapshot) error
	MarkRunning(ctx context.Context, workflowID string) error
	Complete(ctx context.Context, snapshot CompletedWorkflowSnapshot) error
	Fail(ctx context.Context, snapshot FailedWorkflowSnapshot) error
	FindByID(ctx context.Context, userID uint, workflowID string) (WorkflowSnapshot, error)
}
```

方針:
- application は workflow 状態保存 port を先に定義する。
- `TODO:` 履歴テーブル名、カラム設計、保持期間、部分結果の保存粒度は後続で決める。
- 今回は API 契約と application の責務整理を優先する。

## 6. dispatcher 設計

```go
type WorkflowDispatcher interface {
	Dispatch(ctx context.Context, job DispatchJob) error
}

type DispatchJob struct {
	WorkflowID   string
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}
```

初期方針:
- まずは in-process dispatcher でよい。
- 将来 queue 化する場合でも、application は `WorkflowDispatcher` interface を使う。
- request context は dispatch 完了までに閉じ、background 実行では新しい context を起こす。

## 7. stage 接続

### `manualmailworkflow` が background で行う処理
1. `mailfetch` stage を呼ぶ。
2. 返却された `created_emails` を `mailanalysis` stage に渡す。
3. `mailanalysis` stage を呼ぶ。
4. 返却された `parsed_emails` payload を `vendorresolution` stage に渡す。
5. `vendorresolution` stage を呼ぶ。
6. 返却された `resolved_items` を `billingeligibility` stage に渡す。
7. `billingeligibility` stage を呼ぶ。
8. 返却された `eligible_items` を `billing` stage に渡す。
9. `billing` stage を呼ぶ。
10. 各段の結果をまとめて workflow 結果にする。

## 8. `manualmailworkflow` で持つ port

### stage ports
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

### async workflow ports
```go
type WorkflowStatusRepository interface {
	CreateQueued(ctx context.Context, snapshot QueuedWorkflowSnapshot) error
	MarkRunning(ctx context.Context, workflowID string) error
	Complete(ctx context.Context, snapshot CompletedWorkflowSnapshot) error
	Fail(ctx context.Context, snapshot FailedWorkflowSnapshot) error
	FindByID(ctx context.Context, userID uint, workflowID string) (WorkflowSnapshot, error)
}

type WorkflowDispatcher interface {
	Dispatch(ctx context.Context, job DispatchJob) error
}
```

## 9. Adapter 設計

### Adapter 一覧
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
- `InProcessWorkflowDispatcher`
  - background 実行を起動する
- `WorkflowStatusRepositoryAdapter`
  - workflow 状態の read / write を担う

### `InProcessWorkflowDispatcher`
- やること
  - `RunnerUseCase` を background 実行する
  - `workflow_id` を context に積む
- やらないこと
  - stage 業務ロジックの内包
  - queue 製品依存の契約流出

## 10. DI 方針
- DI は `internal/di` に集約する。
- `manualmailworkflow` は stage adapter に加え、dispatcher と status repository を受け取る。
- 将来 queue adapter に差し替える場合も DI だけで切り替えられる構成にする。

## 11. 同期 / 非同期方針
- HTTP は workflow 受付までに責務を限定する。
- workflow の実処理は background で順次実行する。
- `mailfetch` と `mailanalysis` は長時間化しやすいため、非同期化の主対象とする。
- `vendorresolution`, `billingeligibility`, `billing` は background workflow 内では決定的処理として扱う。

## 12. TODO
- workflow 履歴テーブル / 結果保存テーブルの物理設計
- stage ごとの途中経過をどの粒度で永続化するか
- queue 導入時の再実行戦略、lease、timeout 設計
