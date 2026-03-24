# 手動メール取得アーキテクチャ 設計

## 1. 設計方針
- DDD 上の `ManualMailFetch` は「取得」までに責務を限定する。
- AI 解析、Vendor 正規化、請求成立判定、請求保存は別 package に分ける。
- ただし、`mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` の全体フローは別の統合 package で管理する。
- 推奨する統合 package は `internal/manualmailworkflow` とする。
- `manualmailworkflow` はドメイン概念ではなく、application 層の技術的なオーケストレーション package として扱う。
- package 間の依存は次の形にする。
  - `presentation -> manualmailworkflow -> mailfetch / mailanalysis / vendorresolution / billingeligibility / billing`
- 具体的な Gmail / OpenAI / Gorm 実装は各 package の infrastructure に閉じ込める。
- package 間の接続は port + adapter で行う。
- provider 切替点のみ factory を使う。
- DB transaction は stage ごとに短く閉じ、外部 I/O をまたぐ workflow 全体 transaction は採用しない。
- Go では abstract class は使わず、interface + composition を使う。
- 一連の流れを表現するのは adapter ではなく usecase とする。

## 2. 推奨ディレクトリ

### `internal/manualmailworkflow`
- `application`
  - 全体フローの usecase
  - 手動メール取得開始 command の受付
  - fetch / analyze / vendorresolution / billingeligibility / billing の順序制御
  - 全体サマリ生成
  - 全体ログ、全体ステータス管理
- `infrastructure`
  - `mailfetch` の direct adapter
  - `mailanalysis` の direct adapter
  - `vendorresolution` の direct adapter
  - `billingeligibility` の direct adapter
  - `billing` の direct adapter
  - 将来必要なら queue adapter

補足:
- `manualmailworkflow` は orchestration 専用のため、基本的に独自 domain は持たない。

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
  - 保存済み `parsed_email_ids` の返却
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
  - `ParsedEmail` 読み込み
  - `Email` 読み込み
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
  - `ParsedEmailRepository` の Gorm 実装
  - `EmailRepository` の Gorm 実装

### `internal/billingeligibility`
- `application`
  - 請求成立判定 usecase
  - `VendorResolution` の結果受け取り
  - 請求成立条件の評価
  - eligible / ineligible の返却
- `domain`
  - `EligibleItem`
  - `IneligibleItem`
  - `ParsedEmailForEligibility`
  - `Failure`
- `infrastructure`
  - `ParsedEmailRepository` の Gorm 実装

### `internal/billing`
- `application`
  - `Billing` 生成 usecase
  - `BillingEligibility` の結果受け取り
  - 重複確認
  - `Billing` 保存
  - 導出結果サマリ返却
- `domain`
  - 必要なら stage 専用の input / failure model
  - 初期実装では `Billing`, `Money`, `BillingNumber`, `InvoiceNumber`, `PaymentCycle` は `internal/common/domain` を再利用する
- `infrastructure`
  - `BillingRepository` の Gorm 実装

## 3. 全体フロー
1. presentation が手動メール取得 request を受ける。
2. `manualmailworkflow.UseCase.Execute` が全体フローを開始する。
3. `manualmailworkflow` が `mailfetch` を呼び、`Email` を取得・保存する。
4. `manualmailworkflow` が `mailfetch` の返却した `created_emails` を `mailanalysis` に渡す。
5. `manualmailworkflow` が `mailanalysis` を呼び、`ParsedEmail` を解析・保存する。
6. `manualmailworkflow` が保存済み `parsed_email_ids` を `vendorresolution` に渡す。
7. `manualmailworkflow` が `vendorresolution` を呼び、canonical `Vendor` を解決する。
8. `manualmailworkflow` が `vendorresolution` の返却した解決結果を `billingeligibility` に渡す。
9. `manualmailworkflow` が `billingeligibility` を呼び、請求成立可否を判定する。
10. `manualmailworkflow` が `billingeligibility` の返却した成立結果を `billing` に渡す。
11. `manualmailworkflow` が `billing` を呼び、`Billing` を生成・保存する。
12. `manualmailworkflow` が全体結果をまとめて返す。

## 4. package ごとの処理詳細

### `internal/manualmailworkflow` が行う処理
1. 手動メール取得の開始 request を受ける。
2. 全体フロー用 command を組み立てる。
3. `mailfetch` stage を呼ぶ。
4. 返却された `created_emails` を `mailanalysis` stage に渡す。
5. `mailanalysis` stage を呼ぶ。
6. 返却された `parsed_email_ids` 一覧を `vendorresolution` stage に渡す。
7. `vendorresolution` stage を呼ぶ。
8. 返却された `resolved_items` を `billingeligibility` stage に渡す。
9. `billingeligibility` stage を呼ぶ。
10. 返却された `eligible_items` を `billing` stage に渡す。
11. `billing` stage を呼ぶ。
12. 各段の結果をまとめて全体サマリを返す。

補足:
- 各段の順序制御はここで行う。
- 全体フローをここに集約することで、各 stage の中から次段をさらに起動するようなネストを避ける。
- retry 方針、全体ログ、ステップ単位の結果統合もここに置く。

### `internal/mailfetch` が行う処理
1. `user_id` と `connection_id` の整合性を確認する。
2. `FetchCondition` の必須項目を検証する。
3. `MailAccountConnection` から provider 種別と認可情報を解決する。
4. `MailFetcherFactory` で provider に応じた fetcher を生成する。
5. fetcher で外部メールサービスからメッセージ一覧と詳細を取得する。
6. 取得結果を `Email` に変換する。
7. `user + external_message_id` の一意性で idempotent に保存する。
8. 保存済み `created_email_ids` / `created_emails` / `existing_email_ids` を返す。

補足:
- ここでは本文の意味解釈をしない。
- `mailanalysis` の呼び出しは行わない。

### `internal/mailanalysis` が行う処理
1. `mailfetch` から渡された `created_emails` を受け取る。
2. `created_emails` に含まれる件名・送信元・受信日時・本文を使って解析対象を組み立てる。
3. prompt を構築する。
4. `AnalyzerFactory` で利用する AI analyzer を生成する。
5. AI に解析を依頼し、応答を `ParsedEmail` の配列へ変換する。
6. `ExtractedAt` をシステム側で付与する。
7. `ParsedEmail` を `Email` に紐づく解析履歴として保存する。
8. 保存済み `parsed_email_ids` 一覧を返す。

補足:
- 1メールに対して `ParsedEmail` は複数件あり得る。
- `ParsedEmail` は推定結果であり、最終判断を持たない。
- canonical Vendor の解決はしない。

### `internal/vendorresolution` が行う処理
1. `parsed_email_ids` 一覧を読み込む。
2. 紐づく `ParsedEmail` と元 `Email` を取得する。
3. `VendorResolution` を適用して canonical `Vendor` を解決する。
4. unresolved のものは unresolved として返す。
5. resolved のものは `resolved_items` に積んで返す。

補足:
- `VendorResolution` は `ParsedEmail.vendorName` を候補値として扱う。
- `vendors` / `vendor_aliases` を読み、`name_exact -> sender_domain -> sender_name -> subject_keyword -> unresolved` の順で解決する。
- `BillingEligibility` 判定はしない。
- `Billing` は生成しない。

### `internal/billingeligibility` が行う処理
1. `vendorresolution` の返却した `resolved_items` を受け取る。
2. 請求成立判定に必要な `ParsedEmail` 情報を読み込む。
3. `BillingEligibility` を適用して成立条件を評価する。
4. 成立したものを `eligible_items` として返す。
5. 非成立のものは `ineligible` として理由付きで返す。

補足:
- `BillingEligibility` は `VendorResolution` の結果を前提にする。
- canonical Vendor の解決はしない。
- `Billing` は生成しない。

### `internal/billing` が行う処理
1. `billingeligibility` の返却した `eligible_items` を受け取る。
2. `common/domain.NewBilling(...)` で `Billing` を生成する。
3. `user + vendor + billing_number` で重複を確認する。
4. 未登録なら保存し、既存なら duplicate として返す。
5. 作成件数や重複件数を返す。

補足:
- `Billing` の参照元は `Email` であり、`ParsedEmail` ではない。
- `VendorResolution` と `BillingEligibility` はこの package の責務に含めない。

## 5. Flow / UseCase / Adapter / Factory の役割

### 基本ルール
- 一連の流れを表現するのは `manualmailworkflow` の usecase
- `mailfetch` / `mailanalysis` / `vendorresolution` / `billingeligibility` / `billing` は各段の usecase
- adapter は各段の実装を差し込む
- factory は必要な adapter を実行時に選ぶ
- 業務フローを adapter や factory に持たせない

### 役割分担
- `manualmailworkflow.UseCase`
  - 全体手順を表現する
  - どの順で各 stage を呼ぶかを決める
- 各 package の `UseCase`
  - 自分の境界内の処理だけを表現する
- `Port interface`
  - usecase が必要とする機能の境界を表す
- `Adapter`
  - port の具体実装
  - Gmail、OpenAI、Gorm、stage 連携などの境界を担当する
- `Factory`
  - provider や設定に応じて適切な adapter を返す

### やってはいけない構成
- `FlowAdapter` が fetch -> analyze -> vendorresolution -> billingeligibility -> billing をまとめて持つ
- adapter の中で次の adapter を `new` する
- `mailfetch.UseCase` が `mailanalysis` を直接起動する
- `mailanalysis.UseCase` が `vendorresolution` を直接起動する
- `vendorresolution.UseCase` が `billingeligibility` を直接起動する
- `billingeligibility.UseCase` が `billing` を直接起動する
- factory が業務フロー全体を決める

### 例: `manualmailworkflow` usecase
```go
type UseCase struct {
	fetchStage               FetchStage
	analyzeStage             AnalyzeStage
	vendorResolutionStage    VendorResolutionStage
	billingEligibilityStage  BillingEligibilityStage
	billingStage             BillingStage
}

func (uc *UseCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	fetchResult, err := uc.fetchStage.Execute(ctx, FetchCommand{
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		Condition:    cmd.Condition,
	})
	if err != nil {
		return Result{}, err
	}

	analyzeResult, err := uc.analyzeStage.Execute(ctx, AnalyzeCommand{
		UserID: cmd.UserID,
		Emails: fetchResult.CreatedEmails,
	})
	if err != nil {
		return Result{}, err
	}

	vendorResolutionResult, err := uc.vendorResolutionStage.Execute(ctx, VendorResolutionCommand{
		UserID:         cmd.UserID,
		ParsedEmailIDs: analyzeResult.ParsedEmailIDs,
	})
	if err != nil {
		return Result{}, err
	}

	billingEligibilityResult, err := uc.billingEligibilityStage.Execute(ctx, BillingEligibilityCommand{
		UserID:        cmd.UserID,
		ResolvedItems: vendorResolutionResult.ResolvedItems,
	})
	if err != nil {
		return Result{}, err
	}

	billingResult, err := uc.billingStage.Execute(ctx, BillingCommand{
		UserID:        cmd.UserID,
		EligibleItems: billingEligibilityResult.EligibleItems,
	})
	if err != nil {
		return Result{}, err
	}

	return Result{
		Fetch:              fetchResult,
		Analysis:           analyzeResult,
		VendorResolution:   vendorResolutionResult,
		BillingEligibility: billingEligibilityResult,
		Billing:            billingResult,
	}, nil
}
```

## 6. `manualmailworkflow` で持つ port

### `internal/manualmailworkflow/application`
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

### workflow が扱う result
```go
type Result struct {
	Fetch              FetchResult
	Analysis           AnalyzeResult
	VendorResolution   VendorResolutionResult
	BillingEligibility BillingEligibilityResult
	Billing            BillingResult
}
```

## 7. Factory 設計

### 採用方針
- factory は provider 切替点だけに置く。
- 初期実装で factory を使うのは `mailfetch` と `mailanalysis` に限定する。

### `MailFetcherFactory`
- provider に応じて Gmail などの fetcher を返す。

### `AnalyzerFactory`
- analyzer provider に応じて OpenAI などの analyzer を返す。

### factory を使わない箇所
- `vendorresolution`
  - ルールは DB の alias master で切り替えるため、初期は factory 不要
- `billingeligibility`
  - 判定ルールは application/domain に固定で持つ
- `billing`
  - 永続化先は初期は単一 DB 前提

## 8. Adapter 設計

### 採用方針
- adapter は package 境界または外部依存境界だけに置く。
- stage の direct 呼び出しも adapter として明示する。

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
- `GmailMailFetcherAdapter`
  - Gmail API からメールを取得する
- `OpenAIAnalyzerAdapter`
  - OpenAI API で解析する
- `VendorResolverAdapter`
  - `vendors` / `vendor_aliases` を使って canonical Vendor を返す

### 各 adapter の責務制約

#### `DirectManualMailFetchAdapter`
- やること
  - `mailfetch.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - Gmail API を直接叩く
  - 全体フロー制御

#### `DirectMailAnalysisAdapter`
- やること
  - `mailanalysis.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - OpenAI API を直接叩く
  - vendorresolution 起動

#### `DirectVendorResolutionAdapter`
- やること
  - `vendorresolution.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - billingeligibility 起動

#### `DirectBillingEligibilityAdapter`
- やること
  - `billingeligibility.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - billing 起動

#### `DirectBillingAdapter`
- やること
  - `billing.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - vendor 解決
  - 請求成立判定

#### `VendorResolverAdapter`
- やること
  - alias テーブル、送信元ルール、件名ルールなどを使って canonical Vendor を返す
- やらないこと
  - `BillingEligibility` 判定
  - `Billing` 保存

## 9. 推奨 port

### `internal/vendorresolution/application`
```go
type ParsedEmailRepository interface {
	FindForResolution(ctx context.Context, userID, parsedEmailID uint) (ParsedEmailForResolution, error)
}

type EmailRepository interface {
	FindSource(ctx context.Context, userID, emailID uint) (SourceEmail, error)
}

type VendorResolver interface {
	Resolve(ctx context.Context, input VendorResolutionInput) (ResolutionDecision, error)
}
```

### `internal/billingeligibility/application`
```go
type ParsedEmailRepository interface {
	FindForEligibility(ctx context.Context, userID, parsedEmailID uint) (ParsedEmailForEligibility, error)
}
```

### `internal/billing/application`
```go
type BillingRepository interface {
	ExistsByIdentity(ctx context.Context, userID, vendorID uint, billingNumber string) (bool, error)
	Save(ctx context.Context, billing Billing) (uint, error)
}
```

## 10. DI 方針

### 原則
- DI は `internal/di` に集約する。
- workflow は各 stage を interface として受け取る。
- 具体的な Gorm / Gmail / OpenAI 実装は DI で束ねる。

### 登録方針
- `internal/di/mailfetch.go`
  - `mailfetch` 依存を登録する
- `internal/di/mailanalysis.go`
  - `mailanalysis` 依存を登録する
- `internal/di/vendorresolution.go`
  - `vendorresolution` 依存を登録する
- `internal/di/billingeligibility.go`
  - `billingeligibility` 依存を登録する
- `internal/di/billing.go`
  - `billing` 依存を登録する
- `internal/di/manualmailworkflow.go`
  - `DirectManualMailFetchAdapter`
  - `DirectMailAnalysisAdapter`
  - `DirectVendorResolutionAdapter`
  - `DirectBillingEligibilityAdapter`
  - `DirectBillingAdapter`
  - `manualmailworkflow/application.UseCase`

## 11. 同期 / 非同期方針
- `mailfetch`, `mailanalysis`, `vendorresolution`, `billingeligibility`, `billing` は workflow 上では順次実行する。
- 長時間化しやすいのは主に外部 I/O を持つ `mailfetch` と `mailanalysis` である。
- `vendorresolution`, `billingeligibility`, `billing` は DB 中心の決定的処理として扱う。

## 12. 今回の判断
- `VendorResolution` は `billing` package に含めず、`internal/vendorresolution` として独立させる。
- `BillingEligibility` は `vendorresolution` と `billing` の間に独立 stage として置く。
- `billing` は `BillingEligibility` の結果を受け、`Billing` 生成と保存だけを担当する。
- `manualmailworkflow` は `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` の順でオーケストレーションする。
