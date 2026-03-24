# VendorResolution による Vendor 正規化処理 設計

## 1. 設計方針
- `VendorResolution` は `BillingEligibility` や `Billing` 生成から分離し、単独の usecase として扱う。
- package は `internal/vendorresolution` に切り出す。
- `mailanalysis` は引き続き `ParsedEmail` 保存までに責務を限定し、canonical `Vendor` の解決は持たない。
- vendor 解決は保存済みの `ParsedEmail` と `Email` メタデータのみで行う。
  - 本文は使わない。
  - LLM の再呼び出しはしない。
- vendor 解決ルールと自動登録ルールは DDD のモデルとして `internal/common/domain/vendor_resolution.go` に寄せる。
- `internal/vendorresolution` は workflow 用の application と、DB 読み書き専用の repository 群に絞る。
- vendor 未解決時の監査は初期は構造化ログのみとし、snapshot テーブルは作らない。
- 今回のスコープに `BillingEligibility`、重複確認、`Billing` 保存は含めない。

## 2. package 構成

### `internal/common/domain`
- `VendorResolution`
- `VendorResolutionInput`
- `VendorResolutionFetchPlan`
- `VendorAliasCandidate`
- `VendorResolutionFacts`
- `VendorRegistrationAlias`
- `VendorRegistrationPlan`
- `VendorResolutionDecision`
- `VendorResolutionPolicy`

役割:
- vendor 解決ルールの中心モデルを置く。
- repository が引くべき検索条件を組み立てる。
- repository が集めた候補群から 1 回で最終判定する。
- unresolved 時に自動登録用の保存計画を作る。

### `internal/vendorresolution/application`
- `UseCase`
- `Command`, `Result`
- workflow から受け取る `ResolutionTarget`
- `VendorResolutionRepository`
- `VendorRegistrationRepository`

役割:
- workflow 入力の検証
- 1 件ずつの結果集約
- unresolved 監査ログ
- policy と repository の接続

### `internal/vendorresolution/domain`
- stage 専用の failure model / result model
- common/domain の型 alias / 再エクスポート

役割:
- stage 固有の failure code と result shape を持つ。
- 判定本体は持たず、common/domain のモデルを橋渡しする。

### `internal/vendorresolution/infrastructure`
- `VendorResolutionRepository`
- `VendorRegistrationRepository`
- `vendorRecord`, `vendorAliasRecord`

役割:
- `vendors` / `vendor_aliases` の read / write
- application / domain policy が必要とする shape への変換

## 3. データモデル

### `vendors`
- 役割
  - canonical Vendor master
- カラム
  - `id`
  - `name`
  - `normalized_name`
  - `created_at`
  - `updated_at`
- 制約
  - `UNIQUE (normalized_name)`

補足:
- 自動登録で新規 vendor を作るときも `normalized_name` 一意制約を使って冪等に補完する。

### `vendor_aliases`
- 役割
  - canonical Vendor へ寄せるための共通解決ルール
- カラム
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
  - `UNIQUE (vendor_id, alias_type, normalized_value)`

補足:
- 同じ `alias_type + normalized_value` を複数 vendor に登録できるようにする。
- `name_exact` / `sender_domain` / `sender_name` の競合時は policy が `created_at DESC, id DESC` の順で 1 件を選ぶ。
- `subject_keyword` は最長一致優先とし、最長一致が複数 vendor にまたがる場合は unresolved とする。
- 自動登録時は canonical name を `name_exact` alias として同時に補完する。

## 4. common/domain 設計

### `VendorResolutionInput`
```go
type VendorResolutionInput struct {
	CandidateVendorName *string
	Subject             string
	From                string
	To                  []string
}
```

役割:
- workflow から渡された 1 件分の生入力を保持する。
- `Normalize()` で trim や宛先一覧の空要素除去を行う。

### `VendorResolutionFetchPlan`
```go
type VendorResolutionFetchPlan struct {
	NameExactValue    string
	SenderDomainValue string
	SenderNameValue   string
	SubjectValue      string
}
```

役割:
- repository が DB からどの条件で alias 候補を引くべきかをまとめる。
- `VendorResolutionPolicy.BuildFetchPlan` が生成する。

### `VendorAliasCandidate`
```go
type VendorAliasCandidate struct {
	AliasID         uint
	AliasType       string
	AliasValue      string
	NormalizedValue string
	AliasCreatedAt  time.Time
	Vendor          Vendor
}
```

役割:
- alias lookup の 1 件分の候補。
- exact 系の最新選択や subject keyword の競合判定に使う。

### `VendorResolutionFacts`
```go
type VendorResolutionFacts struct {
	NameExactCandidates      []VendorAliasCandidate
	SenderDomainCandidates   []VendorAliasCandidate
	SenderNameCandidates     []VendorAliasCandidate
	SubjectKeywordCandidates []VendorAliasCandidate
}
```

役割:
- repository が集めた候補群を 1 回分の判定材料として束ねる。

### `VendorRegistrationPlan`
```go
type VendorRegistrationPlan struct {
	VendorName           string
	NormalizedVendorName string
	Aliases              []VendorRegistrationAlias
}
```

役割:
- unresolved の candidate vendor 名から保存すべき vendor / alias を表す。
- 初期実装では `name_exact` alias だけを自動補完対象とする。

### `VendorResolutionPolicy`
- `BuildFetchPlan(input VendorResolutionInput) VendorResolutionFetchPlan`
  - `candidate_vendor_name`、`from`、`subject` から repository 用の検索条件を作る。
- `Resolve(facts VendorResolutionFacts) VendorResolutionDecision`
  - `name_exact -> sender_domain -> sender_name -> subject_keyword -> unresolved` の順で 1 回で最終判定する。
- `BuildRegistrationPlan(input VendorResolutionInput, decision VendorResolutionDecision) *VendorRegistrationPlan`
  - unresolved のときだけ candidate vendor 名から自動登録計画を作る。
- `ResolveRegisteredVendor(vendor Vendor) VendorResolutionDecision`
  - 自動登録で確定した vendor を `name_exact` 解決結果へ変換する。

### 判定ルール

#### `name_exact`
- `ParsedEmail.vendorName` を trim + lowercase + 空白圧縮した値で正規化する。
- `vendor_aliases.alias_type = name_exact` かつ `normalized_value` 完全一致の候補群を使う。
- 複数候補がある場合は `created_at DESC, id DESC` の最新を選ぶ。

#### `sender_domain`
- `Email.from` から送信元メールアドレスを抽出する。
- `@` 以降の domain を lowercase 化し、`sender_domain` の候補群を使う。
- 複数候補がある場合は `created_at DESC, id DESC` の最新を選ぶ。

#### `sender_name`
- `Email.from` から表示名を抽出する。
- trim + lowercase + 空白圧縮した値で `sender_name` の候補群を使う。
- 複数候補がある場合は `created_at DESC, id DESC` の最新を選ぶ。

#### `subject_keyword`
- `Email.subject` を trim + lowercase + 空白圧縮した値に正規化する。
- subject に含まれる `subject_keyword` 候補群を使う。
- 最長 keyword を優先する。
- 最長 keyword が複数 vendor にまたがる場合は unresolved とする。
- 同一 vendor 内の複数候補は `created_at DESC, id DESC` の最新を選ぶ。

#### 自動登録
- 既存ルールで unresolved のときだけ candidate vendor 名から登録計画を作る。
- 初期実装で自動登録するのは `Vendor` と `name_exact` alias のみ。
- `sender_domain` / `sender_name` / `subject_keyword` は自動登録しない。

## 5. application 設計

### `Command`
```go
type Command struct {
	UserID       uint
	ParsedEmails []ResolutionTarget
}
```

### `ResolutionTarget`
```go
type ResolutionTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ParsedEmail       commondomain.ParsedEmail
}
```

補足:
- workflow から `ParsedEmail` 実体と source email の必要メタデータをまとめて受け取る。
- `ParsedEmailID` を受けて DB を引き直す方式は採らない。

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

`Stage`:
- `normalize_input`
- `resolve_vendor`
- `register_vendor`

`Code`:
- `invalid_resolution_target`
- `vendor_resolution_failed`
- `vendor_registration_failed`

### `UseCase` の流れ
1. `user_id` と入力を検証する。
2. `ParsedEmails` を順に処理する。
3. `ResolutionTarget` を normalize する。
4. `VendorResolutionPolicy.BuildFetchPlan` で検索条件を作る。
5. `VendorResolutionRepository.FetchFacts` で候補群を取得する。
6. `VendorResolutionPolicy.Resolve` で最終判定する。
7. unresolved の場合は `VendorResolutionPolicy.BuildRegistrationPlan` を呼び、計画があれば `VendorRegistrationRepository.EnsureByPlan` で補完する。
8. それでも unresolved なら件数と `external_message_id` を集約し、warning log を出す。
9. 解決済みなら `ResolvedItems` と `ResolvedCount` を更新する。
10. read/write repository の内部障害は個別 `Failure` として `Result` に積む。

補足:
- `Execute` の `error` は command 不正、依存未設定などの stage 全体失敗に限定する。
- unresolved は業務結果であり `Failures` には含めない。

## 6. port 設計

### `internal/vendorresolution/application`
```go
type VendorResolutionRepository interface {
	FetchFacts(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error)
}

type VendorRegistrationRepository interface {
	EnsureByPlan(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error)
}
```

## 7. infrastructure 設計

### `VendorResolutionRepository`
- `VendorResolutionFetchPlan` に基づいて以下を取得する。
  - `name_exact` 完全一致候補群
  - `sender_domain` 完全一致候補群
  - `sender_name` 完全一致候補群
  - `subject_keyword` 部分一致候補群
- DB からは判定に必要な列だけを読み、`VendorAliasCandidate` へ変換する。
- 判定順序や競合解消は持たない。

### `VendorRegistrationRepository`
- `VendorRegistrationPlan` に従って `vendors` と `vendor_aliases` を補完する。
- `vendors.normalized_name` の一意制約を使って vendor を冪等に upsert する。
- alias は `UNIQUE (vendor_id, alias_type, normalized_value)` を使って冪等に補完する。
- スキーマ長を超える候補は切り詰めず、`nil` を返して未解決扱いに残す。

## 8. `manualmailworkflow` への接続

### 方針
- `manualmailworkflow` に `vendorresolution` stage として接続する。
- `analysis.ParsedEmails` が 0 件なら `vendorresolution` は空結果で終了する。
- `vendorresolution` の出力は将来 `billingeligibility` stage の入力になる。

### `manualmailworkflow/application`
```go
type VendorResolutionCommand struct {
	UserID       uint
	ParsedEmails []ParsedEmail
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
  - `VendorResolutionRepository`
  - `VendorRegistrationRepository`
  - `vendorresolution/application.UseCase`
- `manualmailworkflow` へ接続する場合は `DirectVendorResolutionAdapter` を追加する。

## 10. 今回の判断
- vendor 解決は `ParsedEmail` と `Email` の保存済み具体値だけで行う。
- 判定ロジックと登録計画生成ロジックは `internal/common/domain` に集約する。
- `internal/vendorresolution/infrastructure` は DB read/write に限定する。
- vendor master は `vendors` と `vendor_aliases` に分ける。
- 解決ルールは `name_exact -> sender_domain -> sender_name -> subject_keyword -> unresolved` の順にする。
- alias 重複は許容し、exact 系は `created_at DESC, id DESC` で 1 件を選ぶ。
- `subject_keyword` は最長一致優先で、同長競合が複数 vendor にまたがる場合は unresolved とする。
- unresolved の candidate vendor 名だけを `Vendor` + `name_exact` alias として自動登録する。
- unresolved 監査は初期は構造化ログのみとする。
- ユーザー単位の上書きルールは後続エンハンスに送る。
