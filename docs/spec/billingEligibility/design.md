# BillingEligibility stage 設計

## 1. 設計方針
- `BillingEligibility` は `vendorresolution` と `billing` の間に置く独立 stage とする。
- 判定本体は `internal/common/domain.BillingEligibility` をベースにするが、`billing_date` 任意化に合わせて必須条件を見直す。
- stage 自体は純粋な決定処理に寄せ、DB / AI / 外部メールサービスへのアクセスは持たない。
- 既存実装では `mailanalysis` と `vendorresolution` が workflow payload を使って接続されているため、`billingeligibility` も同じく workflow payload で接続する。
- `internal/vendorresolution` の usecase 契約は極力変えず、`manualmailworkflow` 側の workflow-owned 型を拡張して `ParsedEmail` を次段へ橋渡しする。
- `billing_number` の生成責務はこの stage に持ち込まず、前段で受け取れる前提に固定する。

## 2. package 構成

### `internal/billingeligibility/application`
- `UseCase`
- `Command`, `Result`
- `EligibilityTarget`
- stage 内の入力検証、policy 呼び出し、結果集約

### `internal/billingeligibility/domain`
- `EligibleItem`
- `IneligibleItem`
- `Failure`
- `ReasonCode`
- stage 固有の result / failure shape

### `internal/billingeligibility/infrastructure`
- 初期実装では不要
- `billingeligibility` は純粋な決定処理なので repository を持たない

### `internal/manualmailworkflow`
- `BillingEligibilityStage` 追加
- `BillingEligibilityCommand`, `BillingEligibilityResult` 追加
- `Result` と controller response へ `billing_eligibility` を追加

### `internal/di`
- `billingeligibility.go` を追加
- `manualmailworkflow.go` に `DirectBillingEligibilityAdapter` を登録する

## 3. workflow 境界の設計

### 3.1 `manualmailworkflow/application.ResolvedItem` を拡張する
`vendorresolution` usecase 自体の返り値は現状のままにし、workflow 側の `ResolvedItem` に判定用 `ParsedEmail` を保持させる。

```go
type ResolvedItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	Data              commondomain.ParsedEmail
}
```

理由:
- `vendorresolution` は既に入力として `ParsedEmail` 実体を受け取っている。
- `billingeligibility` が必要とする情報は `ParsedEmail` 内に既に揃っている。
- ここで DB 再読込に戻すと、実装済みの workflow payload パターンから外れ、不要な I/O が増える。

### 3.2 `DirectVendorResolutionAdapter` で `Data` を補完する
adapter は `cmd.ParsedEmails` を持っているため、`parsed_email_id` をキーに元の `ParsedEmail` を引ける。

処理:
1. `cmd.ParsedEmails` から `parsed_email_id -> ParsedEmail` の map を作る。
2. `vendorresolution` usecase の `ResolvedItems` を workflow-owned `ResolvedItem` に変換する。
3. 変換時に `Data` を map から補完する。

この方法なら `internal/vendorresolution` package の public contract を増やさずに済む。

## 4. `billingeligibility` application 設計

### 4.1 `Command`
```go
type Command struct {
	UserID        uint
	ResolvedItems []EligibilityTarget
}
```

### 4.2 `EligibilityTarget`
```go
type EligibilityTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	Data              commondomain.ParsedEmail
}
```

補足:
- 判定に `subject` / `from` / `to` は不要なので持ち込まない。
- `VendorResolution` 自体の再評価はせず、解決済み `Vendor` を前提にする。

### 4.3 `Result`
```go
type Result struct {
	EligibleItems   []domain.EligibleItem
	EligibleCount   int
	IneligibleItems []domain.IneligibleItem
	IneligibleCount int
	Failures        []domain.Failure
}
```

### 4.4 `EligibleItem`
```go
type EligibleItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	BillingNumber     string
	InvoiceNumber     *string
	Amount            float64
	Currency          string
	BillingDate       *time.Time
	PaymentCycle      string
}
```

役割:
- 後続 `billing` stage の入力としてそのまま使える shape にする。
- `Billing` はまだ生成しないが、後続 stage が必要とする `billing_number` はここで必ず保持する。
- `billing_date` だけは任意項目のまま後続へ渡す。

### 4.5 `IneligibleItem`
```go
type IneligibleItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	ReasonCode        string
}
```

### 4.6 `Failure`
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
- `evaluate_eligibility`

`Code`:
- `invalid_eligibility_target`
- `billing_eligibility_failed`

## 5. reason code 設計
`common/domain.BillingEligibility.Evaluate` が返す error は、application で安定した reason code に写像する。

| policy error | reason code |
| --- | --- |
| `ErrBillingEligibilityAmountEmpty` | `amount_empty` |
| `ErrBillingEligibilityAmountInvalid` | `amount_invalid` |
| `ErrBillingEligibilityCurrencyEmpty` | `currency_empty` |
| `ErrBillingEligibilityCurrencyInvalid` | `currency_invalid` |
| `ErrBillingEligibilityBillingNumberEmpty` | `billing_number_empty` |
| `ErrBillingEligibilityPaymentCycleEmpty` | `payment_cycle_empty` |
| `ErrBillingEligibilityPaymentCycleInvalid` | `payment_cycle_invalid` |

方針:
- 上記に写像できるものは業務上の `ineligible` として扱う。
- 写像できない予期しない error のみ `Failure` に積む。
- vendor 未解決は前段責務なので、`vendor_id` / `vendor_name` 欠落は target validation で `invalid_eligibility_target` に寄せる。
- `billing_date` は不在でも non-eligible 理由にしない。

## 6. `UseCase` の流れ
1. `ctx`、`user_id`、依存を検証する。
2. `ResolvedItems` が 0 件なら空結果で終了する。
3. 各 target を順に処理する。
4. target を normalize する。
  - `VendorName` trim
  - `Data.Normalize()`
5. target の最低条件を検証する。
  - `parsed_email_id`
  - `email_id`
  - `vendor_id`
  - `vendor_name`
  - `billing_number`
6. 検証に通ったら、`commondomain.VendorResolution{ResolvedVendor: &commondomain.Vendor{...}}` を組み立てる。
7. `commondomain.BillingEligibility{}.Evaluate(target.Data, resolution)` を呼ぶ。
8. `nil` の場合は `EligibleItem` に変換する。
9. 既知 error の場合は `IneligibleItem` に reason code を付けて追加する。
10. `ErrBillingEligibilityVendorUnresolved` を含む前提外 error は `Failure` に変換する。
11. 件数を集計して返す。

補足:
- `Execute` の top-level `error` は command 不正や nil context などの stage 全体失敗に限定する。
- 不成立は業務結果であり `error` にはしない。
- 実装時は `common/domain.BillingEligibility` から `billing_date` 必須チェックだけを外す想定にする。
- `billing_number` の補完ロジックや digest fallback の生成は、この stage の外で完了している前提にする。

## 7. `manualmailworkflow` への接続

### 7.1 application
`manualmailworkflow/application` に以下を追加する。

```go
type BillingEligibilityCommand struct {
	UserID        uint
	ResolvedItems []ResolvedItem
}

type BillingEligibilityResult struct {
	EligibleItems   []EligibleItem
	EligibleCount   int
	IneligibleItems []IneligibleItem
	IneligibleCount int
	Failures        []BillingEligibilityFailure
}

type BillingEligibilityStage interface {
	Execute(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error)
}
```

`Result` も拡張する。

```go
type Result struct {
	Fetch              FetchResult
	Analysis           AnalyzeResult
	VendorResolution   VendorResolutionResult
	BillingEligibility BillingEligibilityResult
}
```

### 7.2 usecase の実行順
1. fetch
2. analysis
3. vendorresolution
4. billingeligibility

条件:
- `analysis.ParsedEmails == 0` なら `vendorresolution` を skip
- `vendorresolution.ResolvedItems == 0` なら `billingeligibility` を skip

### 7.3 logging
workflow 完了ログに以下を追加する。
- `eligible_billing_count`
- `ineligible_billing_count`
- `billing_eligibility_failure_count`

## 8. direct adapter 設計

### `DirectBillingEligibilityAdapter`
- `billingeligibility.UseCase` を呼ぶ
- workflow-owned `ResolvedItem` を `EligibilityTarget` に変換する
- application の `Result` を workflow-owned `BillingEligibilityResult` に変換する

### `DirectVendorResolutionAdapter` の変更
- 既存の `ResolvedItems` 変換時に `Data` を補完する
- `vendorresolution` usecase の interface 自体は変更しない

## 9. presentation response 設計
`POST /api/v1/manual-mail-workflows` のレスポンスに `billing_eligibility` を追加する。

```json
{
  "message": "メール取得ワークフローが完了しました。",
  "fetch": { ... },
  "analysis": { ... },
  "vendor_resolution": { ... },
  "billing_eligibility": {
    "eligible_count": 1,
    "eligible_items": [
      {
        "parsed_email_id": 9001,
        "email_id": 101,
        "external_message_id": "msg-1",
        "vendor_id": 3001,
        "vendor_name": "Acme",
        "matched_by": "name_exact",
        "billing_number": "digest_8e4b1c...",
        "invoice_number": null,
        "amount": 1200,
        "currency": "JPY",
        "billing_date": null,
        "payment_cycle": "one_time"
      }
    ],
    "ineligible_count": 1,
    "ineligible_items": [
      {
        "parsed_email_id": 9002,
        "email_id": 101,
        "external_message_id": "msg-1",
        "vendor_id": 3001,
        "vendor_name": "Acme",
        "matched_by": "name_exact",
        "reason_code": "currency_empty"
      }
    ],
    "failure_count": 0,
    "failures": []
  }
}
```

方針:
- HTTP では Go の error 文字列を出さない。
- `reason_code` / `code` は `snake_case` の安定値にする。

## 10. DI 方針
- `internal/di/billingeligibility.go` を追加する。
- 登録内容:
  - `billingeligibility/application.UseCase`
- `internal/di/manualmailworkflow.go` を更新する。
  - `DirectBillingEligibilityAdapter`
  - `manualmailworkflow.NewUseCase(...)` に billingeligibility stage を渡す

## 11. テスト方針

### `internal/billingeligibility/application/usecase_test.go`
- eligible になる正常系
- `amount_empty`
- `amount_invalid`
- `currency_empty`
- `currency_invalid`
- `payment_cycle_empty`
- `payment_cycle_invalid`
- `billing_date == nil` でも eligible になること
- `billing_number` が存在すれば digest fallback 由来でも eligible になること
- invalid target を failure にできること
- 複数件で eligible / ineligible / failure が混在すること

### `internal/manualmailworkflow/application/usecase_test.go`
- `vendorresolution` の後に `billingeligibility` が呼ばれること
- resolved item 0 件で `billingeligibility` を skip すること
- 結果に `BillingEligibility` が統合されること

### `internal/manualmailworkflow/infrastructure/direct_vendor_resolution_adapter_test.go`
- `ResolvedItem.Data` が入力 `ParsedEmail` から補完されること

### `internal/manualmailworkflow/infrastructure/direct_billing_eligibility_adapter_test.go`
- command / result 変換が壊れていないこと

### `internal/app/presentation/manualmailworkflow/controller_test.go`
- レスポンスに `billing_eligibility` が含まれること
- `reason_code` / `code` が契約どおり返ること

## 12. 今回の判断
- `BillingEligibility` は pure な application/domain stage とし、repository を持たせない。
- `ParsedEmail` の再読込はせず、workflow payload をそのまま使う。
- `vendorresolution` package 本体の public contract は広げず、workflow adapter で `ParsedEmail` を補完する。
- `eligible_items` は将来の `billing` stage へそのまま渡せる shape にする。
- `VendorResolution` の unresolved は前段責務に留め、`billingeligibility` では「成立/不成立判定」に集中する。
- `billing_date` は `BillingEligibility` の必須条件から外す。
- `billing_number` は必須とし、digest fallback を含めて前段から受け取る前提にする。
