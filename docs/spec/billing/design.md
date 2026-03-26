# Billing stage 設計

## 1. 設計方針
- `billing` は `billingeligibility` の後段に置く独立 stage とする。
- 入力は workflow payload の `eligible_items` をそのまま使い、`Billing` 生成に追加の `ParsedEmail` / `Email` 再読込は持ち込まない。
- 生成は `internal/common/domain.NewBilling(...)` を使い、ドメイン不変条件は `common/domain` に寄せる。
- `BillingLineItem` の正規化と fallback 明細生成も `NewBilling(...)` に寄せ、application では aggregate 初期化を分散させない。
- duplicate 制御は application の事前 `Exists` チェックではなく、repository の idempotent 保存契約と DB 一意制約で守る。
- duplicate は業務結果であり、unexpected error だけを `failure` として返す。

## 2. package 構成

### `internal/billing/application`
- `UseCase`
- `Command`, `Result`
- `CreationTarget`
- `BillingRepository`
- stage 内の入力検証、`Billing` 生成、保存結果の集約

### `internal/billing/domain`
- `CreatedItem`
- `DuplicateItem`
- `Failure`
- stage 固有の result / failure shape

### `internal/billing/infrastructure`
- `BillingRepository` の Gorm 実装
- DB 一意制約違反の duplicate 変換

### `internal/manualmailworkflow`
- `BillingStage` 追加
- `BillingCommand`, `BillingResult` 追加
- workflow-owned `Result` と、将来の状態参照 API 向け controller response へ `billing` を追加

### `internal/di`
- `billing.go` を追加
- `manualmailworkflow.go` に `DirectBillingAdapter` を登録する

## 3. workflow 境界の設計

### 3.1 `manualmailworkflow/application.BillingCommand`
```go
type BillingCommand struct {
	UserID        uint
	EligibleItems []EligibleItem
}
```

### 3.2 `CreationTarget`
```go
type CreationTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	ProductNameDisplay *string
	BillingNumber     string
	InvoiceNumber     *string
	Amount            float64
	Currency          string
	BillingDate       *time.Time
	PaymentCycle      string
	LineItems         []CreationLineItem
}

type CreationLineItem struct {
	ProductNameRaw     *string
	ProductNameDisplay *string
	Amount             *float64
	Currency           *string
}
```

理由:
- `billingeligibility` の `EligibleItem` が、既に `Billing` 生成に必要な情報を揃えている。
- `billing` stage は保存責務に集中し、前段の判定責務へ逆流しない。
- `VendorName` / `MatchedBy` は `Billing` aggregate には不要だが、workflow 結果やログの相関情報として維持する。
- `ProductNameDisplay` は任意入力だが、`Billing` aggregate の表示用商品名として保持するため引き継ぐ。
- `LineItems` は application 層の raw input として受け取り、`NewBilling(...)` 内で `BillingLineItem` に変換する。

## 4. `billing` application 設計

### 4.1 `Command`
```go
type Command struct {
	UserID        uint
	EligibleItems []CreationTarget
}
```

### 4.2 `Result`
```go
type Result struct {
	CreatedItems   []domain.CreatedItem
	CreatedCount   int
	DuplicateItems []domain.DuplicateItem
	DuplicateCount int
	Failures       []domain.Failure
}
```

### 4.3 `CreatedItem`
```go
type CreatedItem struct {
	BillingID         uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
}
```

### 4.4 `DuplicateItem`
```go
type DuplicateItem struct {
	ExistingBillingID uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
}
```

### 4.5 `Failure`
```go
type Failure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
}
```

`Stage`:
- `normalize_input`
- `build_billing`
- `save_billing`

`Code`:
- `invalid_creation_target`
- `billing_construct_failed`
- `billing_persist_failed`

## 5. repository 契約

### 5.1 `BillingRepository`
```go
type SaveResult struct {
	BillingID uint
	Duplicate bool
}

type BillingRepository interface {
	SaveIfAbsent(ctx context.Context, billing commondomain.Billing) (SaveResult, error)
}
```

方針:
- `ExistsByIdentity` と `Save` を分けない。
- repository は「その identity の Billing が無ければ作る、あれば duplicate として返す」を 1 操作として提供する。
- duplicate の場合でも既存 `billing_id` を返す。

### 5.2 infrastructure の期待動作
1. `user_id + vendor_id + billing_number` の DB 一意制約を前提にする。
2. insert を試みる。
3. 成功したら `Duplicate=false` と新規 `billing_id` を返す。
4. 一意制約違反なら identity で既存 row を引き、`Duplicate=true` と既存 `billing_id` を返す。

これにより concurrent request でも duplicate を安全に業務結果へ落とし込める。

## 6. `UseCase` の流れ
1. `ctx`、`user_id`、依存を検証する。
2. `EligibleItems` が 0 件なら空結果で終了する。
3. 各 target を normalize する。
4. target の最低条件を検証する。
  - `parsed_email_id`
  - `email_id`
  - `vendor_id`
  - `billing_number`
  - `currency`
  - `payment_cycle`
5. `commondomain.NewBilling(...)` を呼び、raw line item input の変換・`BillingLineItem` の正規化・fallback 補完を含む `Billing` aggregate を生成する。
6. repository の `SaveIfAbsent` を呼び、`Billing` とその `billing.LineItems` を同一作成操作で保存する。
7. `Duplicate=false` なら `CreatedItem` に積む。
8. `Duplicate=true` なら `DuplicateItem` に積む。
9. 予期しない構築失敗や永続化失敗は `Failure` に積む。
10. 件数を集計して返す。

補足:
- top-level `error` は command 不正や nil context など stage 全体失敗に限定する。
- duplicate は `error` にしない。
- `billing_date == nil` は `NewBilling` 側で許容される前提にする。

## 7. `manualmailworkflow` への接続

### 7.1 application
```go
type BillingResult struct {
	CreatedItems   []CreatedItem
	CreatedCount   int
	DuplicateItems []DuplicateItem
	DuplicateCount int
	Failures       []BillingFailure
}

type BillingStage interface {
	Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}
```

workflow 全体 result は以下に拡張する。

```go
type Result struct {
	Fetch              FetchResult
	Analysis           AnalyzeResult
	VendorResolution   VendorResolutionResult
	BillingEligibility BillingEligibilityResult
	Billing            BillingResult
}
```

### 7.2 usecase の実行順
1. fetch
2. analysis
3. vendorresolution
4. billingeligibility
5. billing

条件:
- `analysis.ParsedEmails == 0` なら `vendorresolution` を skip
- `vendorresolution.ResolvedItems == 0` なら `billingeligibility` を skip
- `billingeligibility.EligibleItems == 0` なら `billing` を skip

### 7.3 logging
workflow 完了ログに以下を追加する。
- `created_billing_count`
- `duplicate_billing_count`
- `billing_failure_count`

## 8. direct adapter 設計

### `DirectBillingAdapter`
- やること
  - `billing.UseCase` を呼ぶ
  - workflow-owned 型と `billing` package の型を相互変換する
- やらないこと
  - `BillingEligibility` 再判定
  - Gorm の直接操作
  - 全体フロー制御

変換方針:
- `manualmailworkflow.EligibleItem` -> `billing/application.CreationTarget`
- `billing/application.Result` -> `manualmailworkflow.BillingResult`

## 9. テスト観点

### `internal/billing/application/usecase_test.go`
- 新規作成時に `CreatedItem` が返ること
- 同一 identity が既に存在する場合に `DuplicateItem` が返ること
- `billing_date == nil` でも保存できること
- 不正な入力が `Failure` に分類されること
- repository error が部分 failure として集約されること

### `internal/billing/infrastructure/repository_test.go`
- DB 一意制約違反を duplicate へ変換できること
- 競合時でも 1 件だけが created になり、残りが duplicate になること

### `internal/manualmailworkflow`
- `billingeligibility` の後に `billing` が呼ばれること
- eligible item 0 件なら `billing` を skip すること
- workflow 統合 result に `billing` 要約が載ること
- 状態参照 API を追加する場合は controller response に `billing` 要約が載ること
