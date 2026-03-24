# VendorResolution による Vendor 正規化処理 設計

## 1. 設計方針
- `VendorResolution` と `BillingEligibility` は `internal/billing` に置く。
- `mailanalysis` は引き続き `ParsedEmail` 保存までに責務を限定し、canonical `Vendor` の解決は持たない。
- vendor 解決は保存済みの `ParsedEmail` と `Email` メタデータのみで行う。
  - 本文は使わない。
  - LLM の再呼び出しはしない。
- 解決ルールは全体共通ルールのみを対象にし、ユーザー単位の上書きは扱わない。
- vendor 未解決時の監査は初期は構造化ログのみとし、snapshot テーブルは作らない。
- `billing.UseCase.Execute` は、vendor 未解決だったメールの `external_message_id` 一覧を result に含める。

## 2. package 構成

### `internal/billing/application`
- `UseCase`
- `Command`, `Result`
- port interface
- `ParsedEmail` 読み込み
- `Email` 読み込み
- `VendorResolution` 適用
- `BillingEligibility` 適用
- 重複確認
- `Billing` 保存

### `internal/billing/domain`
- stage 専用の read model / failure model / input model
- `VendorResolutionInput`
- `ParsedEmailForBilling`
- `SourceEmail`
- `Failure`

補足:
- 初期実装では既存の `Vendor`, `VendorResolution`, `BillingEligibility`, `Billing`, `Money`, `PaymentCycle` は `internal/common/domain` を再利用する。
- 既存 domain を billing package へ大きく移し替えるリファクタは今回スコープに入れない。

### `internal/billing/infrastructure`
- `VendorResolverAdapter`
- `GormParsedEmailReaderAdapter`
- `GormSourceEmailReaderAdapter`
- `GormBillingRepositoryAdapter`
- vendor master / alias の Gorm 読み出し

## 3. データモデル

### `vendors`
- 役割
  - canonical Vendor master
- カラム案
  - `id`
  - `name`
  - `normalized_name`
  - `created_at`
  - `updated_at`
- 制約
  - `UNIQUE (normalized_name)`

補足:
- canonical name 自体でも解決できるよう、vendor 作成時に canonical name を `vendor_aliases` に `name_exact` として登録する。

### `vendor_aliases`
- 役割
  - canonical Vendor へ寄せるための共通解決ルール
- カラム案
  - `id`
  - `vendor_id`
  - `alias_type`
  - `alias_value`
  - `normalized_value`
  - `created_at`
  - `updated_at`
- `alias_type` 値
  - `name_exact`
  - `sender_domain`
  - `sender_name`
  - `subject_keyword`
- 制約
  - `INDEX (vendor_id)`
  - `UNIQUE (vendor_id, alias_type, normalized_value)` を基本とする

補足:
- 同じ `alias_type + normalized_value` を複数 vendor に登録できるようにする。
- resolver は `created_at DESC, id DESC` の順で 1 件を選ぶ。
- `subject_keyword` は 1 subject に複数ヒットし得るため、実装では最長一致優先のあと、同長競合時は `created_at DESC, id DESC` で 1 件を選ぶ。

### `billings`
- 役割
  - 請求 aggregate 永続化
- カラム案
  - `id`
  - `user_id`
  - `vendor_id`
  - `email_id`
  - `billing_number`
  - `invoice_number`
  - `amount`
  - `currency`
  - `billing_date`
  - `payment_cycle`
  - `created_at`
  - `updated_at`
- 制約
  - `UNIQUE (user_id, vendor_id, billing_number)`
  - `INDEX (user_id, email_id)`
  - `INDEX (vendor_id)`

## 4. resolution ルール

### 入力情報
`VendorResolution` は以下を入力に使う。
- `ParsedEmail.vendorName`
- `Email.subject`
- `Email.from`
- `Email.to`
- `Email.external_message_id`

### 優先順位
1. `name_exact`
2. `sender_domain`
3. `sender_name`
4. `subject_keyword`
5. unresolved

### 具体ルール

#### `name_exact`
- `ParsedEmail.vendorName` を trim + lowercase + 空白圧縮した値で正規化する。
- `vendor_aliases.alias_type = name_exact` かつ `normalized_value` 完全一致で探す。
- 複数ヒット時は `created_at DESC, id DESC` で 1 件を選び、その `vendor_id` を返す。

#### `sender_domain`
- `Email.from` から送信元メールアドレスを抽出する。
- `@` 以降の domain を lowercase 化して `sender_domain` alias と完全一致で探す。
- 複数ヒット時は `created_at DESC, id DESC` で 1 件を選ぶ。

#### `sender_name`
- `Email.from` から表示名を抽出する。
- trim + lowercase + 空白圧縮した値で `sender_name` alias を完全一致検索する。
- 複数ヒット時は `created_at DESC, id DESC` で 1 件を選ぶ。

#### `subject_keyword`
- `Email.subject` を trim + lowercase + 空白圧縮した値に正規化する。
- `subject_keyword` alias の `normalized_value` が subject に含まれるかで判定する。
- 複数ヒット時は最長 keyword を優先する。
- 最長 keyword が複数 vendor にまたがる場合は、その集合の中で `created_at DESC, id DESC` で 1 件を選ぶ。

#### unresolved
- どの規則でも解決できなければ unresolved とする。
- 構造化ログへ以下を出す。
  - `user_id`
  - `parsed_email_id`
  - `email_id`
  - `external_message_id`
  - `candidate_vendor_name`
  - `from`
  - `subject`

## 5. application 設計

### `Command`
```go
type Command struct {
	UserID         uint
	ParsedEmailIDs []uint
}
```

### `Result`
```go
type Result struct {
	CreatedBillingIDs            []uint
	CreatedCount                 int
	DuplicateCount               int
	IneligibleCount              int
	UnresolvedCount              int
	UnresolvedExternalMessageIDs []string
	Failures                     []Failure
}
```

補足:
- `UnresolvedCount` は unresolved になった `ParsedEmail` 件数を表す。
- `UnresolvedExternalMessageIDs` は unresolved になったメールの `external_message_id` を重複除去して保持する。

### `Failure`
```go
type Failure struct {
	ParsedEmailID      uint
	EmailID            uint
	ExternalMessageID  string
	Stage              string
	Code               string
}
```

`Stage` 例:
- `load_parsed_email`
- `load_source_email`
- `resolve_vendor`
- `evaluate_billing`
- `check_duplicate`
- `save_billing`

`Code` 例:
- `parsed_email_not_found`
- `source_email_not_found`
- `vendor_unresolved`
- `vendor_resolution_failed`
- `billing_ineligible`
- `billing_duplicate`
- `billing_save_failed`

### `UseCase` の流れ
1. `user_id` と `parsed_email_ids` を検証する。
2. `parsed_email_ids` を順に処理する。
3. `ParsedEmailRepository` で `ParsedEmailForBilling` を取得する。
4. `EmailRepository` で元 `SourceEmail` を取得する。
5. `VendorResolver` に `VendorResolutionInput` を渡す。
6. unresolved の場合:
  - `UnresolvedCount` を加算する。
  - `UnresolvedExternalMessageIDs` に `external_message_id` を重複なく追加する。
  - `Failures` に `vendor_unresolved` を積む。
  - 構造化ログを出して次へ進む。
7. 解決済みの場合、`BillingEligibility` を評価する。
8. 非成立なら `IneligibleCount` を加算し、`Failures` に `billing_ineligible` を積んで次へ進む。
9. 成立なら `common/domain.NewBilling(...)` で `Billing` を生成する。
10. `BillingRepository.ExistsByIdentity(user_id, vendor_id, billing_number)` で重複確認する。
11. 重複時は `DuplicateCount` を加算し、`Failures` に `billing_duplicate` を積んで次へ進む。
12. 未登録なら保存し、`CreatedBillingIDs` と `CreatedCount` を更新する。
13. 全件処理後に `Result` を返す。

補足:
- unresolved / ineligible / duplicate は stage 失敗ではなく業務上の非作成結果として扱う。
- `Execute` の `error` は command 不正、依存未設定などの stage 全体失敗に限定する。

## 6. port 設計

### `internal/billing/application`
```go
type ParsedEmailRepository interface {
	FindForDerivation(ctx context.Context, userID, parsedEmailID uint) (domain.ParsedEmailForBilling, error)
}

type EmailRepository interface {
	FindSource(ctx context.Context, userID, emailID uint) (domain.SourceEmail, error)
}

type VendorResolver interface {
	Resolve(ctx context.Context, input domain.VendorResolutionInput) (commondomain.VendorResolution, error)
}

type BillingRepository interface {
	ExistsByIdentity(ctx context.Context, userID, vendorID uint, billingNumber string) (bool, error)
	Save(ctx context.Context, billing commondomain.Billing) (uint, error)
}
```

### `VendorResolutionInput`
```go
type VendorResolutionInput struct {
	UserID             uint
	ParsedEmailID      uint
	EmailID            uint
	ExternalMessageID  string
	CandidateVendorName *string
	Subject            string
	From               string
	To                 []string
}
```

### `ParsedEmailForBilling`
```go
type ParsedEmailForBilling struct {
	ParsedEmailID      uint
	EmailID            uint
	VendorName         *string
	BillingNumber      *string
	InvoiceNumber      *string
	Amount             *float64
	Currency           *string
	BillingDate        *time.Time
	PaymentCycle       *string
}
```

### `SourceEmail`
```go
type SourceEmail struct {
	EmailID            uint
	ExternalMessageID  string
	Subject            string
	From               string
	To                 []string
	ReceivedAt         time.Time
}
```

## 7. infrastructure 設計

### `VendorResolverAdapter`
- `Resolve` の中で以下を順に実行する。
  - `name_exact` lookup
  - `sender_domain` lookup
  - `sender_name` lookup
  - `subject_keyword` lookup
- lookup 成功時は `common/domain.VendorResolution{ResolvedVendor: &Vendor{...}}` を返す。
- unresolved 時は `ResolvedVendor=nil` で返す。
- DB 読み出し障害は `error` を返す。

### `GormParsedEmailReaderAdapter`
- `parsed_emails` から user-owned な record を 1 件取得する。
- 読み出すのは billing 生成に必要な列だけに限定する。

### `GormSourceEmailReaderAdapter`
- `emails` から user-owned な source email を 1 件取得する。
- `to_json` は `[]string` に decode する。

### `GormBillingRepositoryAdapter`
- `ExistsByIdentity` は `user_id + vendor_id + billing_number` で存在確認する。
- `Save` は `billings` に insert し、採番 ID を返す。

## 8. `manualmailworkflow` への接続

### `manualmailworkflow/application`
- `BillingCommand`
```go
type BillingCommand struct {
	UserID         uint
	ParsedEmailIDs []uint
}
```

- `BillingResult`
```go
type BillingResult struct {
	CreatedBillingIDs            []uint
	CreatedCount                 int
	DuplicateCount               int
	IneligibleCount              int
	UnresolvedCount              int
	UnresolvedExternalMessageIDs []string
	Failures                     []BillingFailure
}
```

- `Result`
  - 既存の `Fetch`, `Analysis` に加えて `Billing` を追加する。

### workflow の流れ
1. `fetch`
2. `analysis`
3. `analysis.ParsedEmailIDs` が 0 件なら `billing` は空結果で終了
4. `billing`

### `billing` response で返すべき要素
- `created_billing_count`
- `created_billing_ids`
- `duplicate_count`
- `ineligible_count`
- `unresolved_count`
- `unresolved_external_message_ids`
- `failure_count`
- `failures`

## 9. DI 方針
- `internal/di/billing.go` を追加する。
- `ProvideBillingDependencies` で以下を登録する。
  - `GormParsedEmailReaderAdapter`
  - `GormSourceEmailReaderAdapter`
  - `VendorResolverAdapter`
  - `GormBillingRepositoryAdapter`
  - `billing/application.UseCase`
- `internal/di/manualmailworkflow.go` は `DirectBillingAdapter` を追加し、`manualmailworkflow.NewUseCase(fetch, analysis, billing, log)` に更新する。

## 10. 今回の判断
- vendor 解決は `ParsedEmail` と `Email` の保存済み具体値だけで行う。
- vendor master は `vendors` と `vendor_aliases` に分ける。
- 解決ルールは `name_exact -> sender_domain -> sender_name -> subject_keyword -> unresolved` の順にする。
- alias 重複は許容し、resolver は `created_at DESC, id DESC` で 1 件を選ぶ。
- `subject_keyword` は最長一致優先のあと、同長競合時も `created_at DESC, id DESC` で 1 件を選ぶ。
- unresolved 監査は初期は構造化ログのみとする。
- `billing.UseCase.Execute` と `manualmailworkflow` の result は `unresolved_external_message_ids` を返す。
- ユーザー単位の上書きルールは後続エンハンスに送る。
