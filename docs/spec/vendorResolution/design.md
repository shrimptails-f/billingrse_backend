# VendorResolution による Vendor 正規化処理 設計

## 1. 設計方針
- `VendorResolution` は `BillingEligibility` や `Billing` 生成から分離し、単独の usecase として扱う。
- package は `internal/vendorresolution` に切り出す。
- `mailanalysis` は引き続き `ParsedEmail` 保存までに責務を限定し、canonical `Vendor` の解決は持たない。
- vendor 解決は保存済みの `ParsedEmail` と `Email` メタデータのみで行う。
  - 本文は使わない。
  - LLM の再呼び出しはしない。
- 解決ルールは全体共通ルールのみを対象にし、ユーザー単位の上書きは扱わない。
- vendor 未解決時の監査は初期は構造化ログのみとし、snapshot テーブルは作らない。
- 今回のスコープに `BillingEligibility`、重複確認、`Billing` 保存は含めない。

## 2. package 構成

### `internal/vendorresolution/application`
- `UseCase`
- `Command`, `Result`
- port interface
- `ParsedEmail` 読み込み
- `Email` 読み込み
- `VendorResolution` 適用
- 解決結果サマリ返却

### `internal/vendorresolution/domain`
- stage 専用の read model / failure model / result model / input model
- `VendorResolutionInput`
- `ParsedEmailForResolution`
- `SourceEmail`
- `ResolutionDecision`
- `ResolvedItem`
- `Failure`

補足:
- 初期実装では既存の `Vendor` と `VendorResolution` は `internal/common/domain` を再利用する。
- 既存 domain を大きく移し替えるリファクタは今回スコープに入れない。

### `internal/vendorresolution/infrastructure`
- `VendorResolverAdapter`
- `GormParsedEmailReaderAdapter`
- `GormSourceEmailReaderAdapter`
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
- `name_exact` / `sender_domain` / `sender_name` の競合時は `created_at DESC, id DESC` の順で 1 件を選ぶ。
- `subject_keyword` は最長一致優先とし、最長一致が複数 vendor にまたがる場合は unresolved とする。

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
- 最長 keyword が複数 vendor にまたがる場合は unresolved とする。
- 最長 keyword が同一 vendor 内で複数 alias に当たる場合は `created_at DESC, id DESC` で 1 件を選ぶ。

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
	ResolvedItems                []ResolvedItem
	ResolvedCount                int
	UnresolvedCount              int
	UnresolvedExternalMessageIDs []string
	Failures                     []Failure
}
```

補足:
- `ResolvedItems` は解決済み `parsed_email_id -> vendor` の対応を返す。
- `UnresolvedCount` は unresolved になった `ParsedEmail` 件数を表す。
- `UnresolvedExternalMessageIDs` は unresolved になったメールの `external_message_id` を重複除去して保持する。
- unresolved は業務上の結果であり、`Failures` には含めない。

### `ResolvedItem`
```go
type ResolvedItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
}
```

### `Failure`
```go
type Failure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
}
```

`Stage` 例:
- `load_parsed_email`
- `load_source_email`
- `resolve_vendor`

`Code` 例:
- `parsed_email_not_found`
- `source_email_not_found`
- `vendor_resolution_failed`

### `UseCase` の流れ
1. `user_id` と `parsed_email_ids` を検証する。
2. `parsed_email_ids` を順に処理する。
3. `ParsedEmailRepository` で `ParsedEmailForResolution` を取得する。
4. `EmailRepository` で元 `SourceEmail` を取得する。
5. `VendorResolver` に `VendorResolutionInput` を渡す。
6. unresolved の場合:
  - `UnresolvedCount` を加算する。
  - `UnresolvedExternalMessageIDs` に `external_message_id` を重複なく追加する。
  - 構造化ログを出して次へ進む。
7. 解決済みの場合:
  - `ResolvedItems` に `parsed_email_id -> vendor` を追加する。
  - `ResolvedCount` を加算する。
8. 全件処理後に `Result` を返す。

補足:
- `Execute` の `error` は command 不正、依存未設定などの stage 全体失敗に限定する。
- `ParsedEmail` 未存在、`Email` 未存在、resolver 内部障害は個別 `Failure` として `Result` に積む。

## 6. port 設計

### `internal/vendorresolution/application`
```go
type ParsedEmailRepository interface {
	FindForResolution(ctx context.Context, userID, parsedEmailID uint) (domain.ParsedEmailForResolution, error)
}

type EmailRepository interface {
	FindSource(ctx context.Context, userID, emailID uint) (domain.SourceEmail, error)
}

type VendorResolver interface {
	Resolve(ctx context.Context, input domain.VendorResolutionInput) (domain.ResolutionDecision, error)
}
```

### `VendorResolutionInput`
```go
type VendorResolutionInput struct {
	UserID              uint
	ParsedEmailID       uint
	EmailID             uint
	ExternalMessageID   string
	CandidateVendorName *string
	Subject             string
	From                string
	To                  []string
}
```

### `ParsedEmailForResolution`
```go
type ParsedEmailForResolution struct {
	ParsedEmailID uint
	EmailID       uint
	VendorName    *string
}
```

### `SourceEmail`
```go
type SourceEmail struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
}
```

### `ResolutionDecision`
```go
type ResolutionDecision struct {
	Resolution commondomain.VendorResolution
	MatchedBy  string
}
```

## 7. infrastructure 設計

### `VendorResolverAdapter`
- `Resolve` の中で以下を順に実行する。
  - `name_exact` lookup
  - `sender_domain` lookup
  - `sender_name` lookup
  - `subject_keyword` lookup
- lookup 成功時は `ResolutionDecision{Resolution: VendorResolution{ResolvedVendor: &Vendor{...}}, MatchedBy: ...}` を返す。
- unresolved 時は `ResolvedVendor=nil` と `MatchedBy=""` を返す。
- DB 読み出し障害は `error` を返す。

### `GormParsedEmailReaderAdapter`
- `parsed_emails` から user-owned な record を 1 件取得する。
- 読み出すのは vendor 解決に必要な列だけに限定する。

### `GormSourceEmailReaderAdapter`
- `emails` から user-owned な source email を 1 件取得する。
- `to_json` は `[]string` に decode する。

## 8. `manualmailworkflow` への接続

### 方針
- `manualmailworkflow` に接続する場合は `vendorresolution` stage として追加する。
- `analysis.ParsedEmailIDs` が 0 件なら `vendorresolution` は空結果で終了する。
- `vendorresolution` の出力は `billingeligibility` stage の入力になる。

### `manualmailworkflow/application`
```go
type VendorResolutionCommand struct {
	UserID         uint
	ParsedEmailIDs []uint
}

type VendorResolutionResult struct {
	ResolvedItems                []ResolvedItem
	ResolvedCount                int
	UnresolvedCount              int
	UnresolvedExternalMessageIDs []string
	Failures                     []VendorResolutionFailure
}
```

### response で返すべき要素
- `resolved_count`
- `unresolved_count`
- `unresolved_external_message_ids`
- `failure_count`
- `failures`

## 9. DI 方針
- `internal/di/vendorresolution.go` を追加する。
- `ProvideVendorResolutionDependencies` で以下を登録する。
  - `GormParsedEmailReaderAdapter`
  - `GormSourceEmailReaderAdapter`
  - `VendorResolverAdapter`
  - `vendorresolution/application.UseCase`
- `manualmailworkflow` へ接続する場合は `DirectVendorResolutionAdapter` を追加する。

## 10. 今回の判断
- vendor 解決は `ParsedEmail` と `Email` の保存済み具体値だけで行う。
- vendor master は `vendors` と `vendor_aliases` に分ける。
- 解決ルールは `name_exact -> sender_domain -> sender_name -> subject_keyword -> unresolved` の順にする。
- alias 重複は許容し、`name_exact` / `sender_domain` / `sender_name` は `created_at DESC, id DESC` で 1 件を選ぶ。
- `subject_keyword` は最長一致優先で、同長競合が複数 vendor にまたがる場合は unresolved とする。
- unresolved 監査は初期は構造化ログのみとする。
- ユーザー単位の上書きルールは後続エンハンスに送る。
