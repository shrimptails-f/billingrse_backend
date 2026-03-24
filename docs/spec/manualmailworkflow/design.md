# 手動メール取得アーキテクチャ 設計

## 1. 設計方針
- DDD 上の `ManualMailFetch` は「取得」までに責務を限定する。
- AI 解析と請求生成は別 package に分ける。
- ただし、`mailfetch -> mailanalysis -> billing` の全体フローは別の統合 package で管理する。
- 推奨する統合 package は `internal/manualmailworkflow` とする。
- `manualmailworkflow` はドメイン概念ではなく、application 層の技術的なオーケストレーション package として扱う。
- package 間の依存は次の形にする。
  - `presentation -> manualmailworkflow -> mailfetch / mailanalysis / billing`
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
  - fetch / analyze / billing の順序制御
  - 全体サマリ生成
  - 全体ログ、全体ステータス管理
- `infrastructure`
  - `mailfetch` の direct adapter
  - `mailanalysis` の direct adapter
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
  - `FetchCondition` をここに置くか、将来 shared package に切り出すかは別途判断
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

### `internal/billing`
- `application`
  - `Billing` 導出 usecase
  - `VendorResolution` 適用
  - `BillingEligibility` 適用
  - 重複判定
  - `Billing` 保存
  - 導出結果サマリ返却
- `domain`
  - `Vendor`
  - `VendorResolution`
  - `Billing`
  - `BillingEligibility`
  - `Money`
  - `BillingNumber`
  - `InvoiceNumber`
  - `PaymentCycle`
- `infrastructure`
  - vendor 解決 adapter
  - `BillingRepository` の Gorm 実装
  - 必要なら判定監査ログ adapter

## 3. 全体フロー
1. presentation が手動メール取得 request を受ける。
2. `manualmailworkflow.UseCase.Execute` が全体フローを開始する。
3. `manualmailworkflow` が `mailfetch` を呼び、`Email` を取得・保存する。
4. `manualmailworkflow` が `mailfetch` の返却した `created_emails` を `mailanalysis` に渡す。
5. `manualmailworkflow` が `mailanalysis` を呼び、`ParsedEmail` を解析・保存する。
6. `manualmailworkflow` が保存済み `parsed_email_ids` を `billing` に渡す。
7. `manualmailworkflow` が `billing` を呼び、`Billing` を生成・保存する。
8. `manualmailworkflow` が全体結果をまとめて返す。

## 4. package ごとの処理詳細

### `internal/manualmailworkflow` が行う処理
1. 手動メール取得の開始 request を受ける。
2. 全体フロー用 command を組み立てる。
3. `mailfetch` stage を呼ぶ。
4. 返却された `created_emails` を `mailanalysis` stage に渡す。
5. `mailanalysis` stage を呼ぶ。
6. 返却された `parsed_email_ids` 一覧を `billing` stage に渡す。
7. `billing` stage を呼ぶ。
8. 各段の結果をまとめて全体サマリを返す。

補足:
- 各段の順序制御はここで行う。
- 全体フローをここに集約することで、`mailfetch` の中から `mailanalysis` をさらに起動するようなネストを避ける。
- retry 方針、全体ログ、ステップ単位の結果統合もここに置く。

### `internal/mailfetch` が行う処理
1. `user_id` と `connection_id` の整合性を確認する。
2. `FetchCondition` の必須項目を検証する。
3. `MailAccountConnection` から provider 種別と認可情報を解決する。
4. `MailFetcherFactory` で provider に応じた fetcher を生成する。
5. fetcher で外部メールサービスからメッセージ一覧と詳細を取得する。
6. 取得結果を `Email` に変換する。
7. `user + external_message_id` の一意性で idempotent に保存する。
8. 保存処理は 20 件単位の batch insert 後に `created_run_id` で created/existing を判定する。
9. 新規保存できた message は、保存済み `email_id` と fetch 済み本文を束ねた `created_emails` に積む。
10. batch insert が失敗した chunk は、その chunk 内の message を `save` failure として扱い、後続 chunk は継続する。
11. `created_email_ids` / `created_emails` / `existing_email_ids` を返す。

補足:
- ここでは本文の意味解釈をしない。
- 「請求メールかどうか」の判定もしない。
- 保存対象は `Email` だけに限定する。
- `mailanalysis` の呼び出しは行わない。
- `manualmailworkflow` は通常 `created_emails` を次段へ流し、`created_email_ids` は保存済み Email の参照に使う。

### `internal/mailanalysis` が行う処理
1. `mailfetch` から渡された `created_emails` を受け取る。
2. `created_emails` に含まれる件名・送信元・受信日時・本文を使って解析対象を組み立てる。
3. prompt を構築する。
4. `AnalyzerFactory` で利用する AI analyzer を生成する。
5. AI に解析を依頼し、応答を `ParsedEmail` の配列へ変換する。
6. `ExtractedAt` をシステム側で付与する。
7. 応答フォーマット不正や空応答を扱う。
8. `ParsedEmail` を `Email` に紐づく解析履歴として保存する。
9. 保存済み `parsed_email_ids` 一覧を返す。

補足:
- 1メールに対して `ParsedEmail` は複数件あり得る。
- `ParsedEmail` は推定結果であり、最終判断を持たない。
- canonical Vendor の解決はしない。
- 保存対象は `ParsedEmail` だけに限定する。
- `billing` の呼び出しは行わない。

### `internal/billing` が行う処理
1. `parsed_email_ids` 一覧を読み込む。
2. 紐づく `ParsedEmail` と元 `Email` を取得する。
3. `VendorResolution` を適用して canonical Vendor を解決する。
4. vendor 未解決なら `Billing` を作らず終了する。
5. `BillingEligibility` を適用して成立条件を評価する。
6. 非成立なら `Billing` を作らず終了する。
7. 成立した場合だけ `Billing` を生成する。
8. `user + vendor + billing_number` で重複を確認する。
9. 未登録なら保存し、既存ならスキップする。
10. 作成件数やスキップ件数を返す。

補足:
- `VendorResolution` と `BillingEligibility` は billing 側の判断責務。
- `Billing` の参照元は `Email` であり、`ParsedEmail` ではない。
- `VendorResolution` / `BillingEligibility` 自体は原則として永続化対象ではない。
- 監査が必要なら結果スナップショットを別テーブルまたはログに保存する。

## 5. Flow / UseCase / Adapter / Factory の役割

### 基本ルール
- 一連の流れを表現するのは `manualmailworkflow` の usecase
- `mailfetch` / `mailanalysis` / `billing` は各段の usecase
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
- `FlowAdapter` が fetch -> analyze -> billing をまとめて持つ
- adapter の中で次の adapter を `new` する
- `mailfetch.UseCase` が `mailanalysis` を直接起動する
- `mailanalysis.UseCase` が `billing` を直接起動する
- factory が業務フロー全体を決める

### 例: `manualmailworkflow` usecase
```go
type UseCase struct {
	fetchStage   FetchStage
	analyzeStage AnalyzeStage
	billingStage BillingStage
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

	billingResult, err := uc.billingStage.Execute(ctx, BillingCommand{
		UserID:         cmd.UserID,
		ParsedEmailIDs: analyzeResult.ParsedEmailIDs,
	})
	if err != nil {
		return Result{}, err
	}

	return Result{
		EmailIDs:       fetchResult.CreatedEmailIDs,
		ParsedEmailIDs: analyzeResult.ParsedEmailIDs,
		BillingSummary: billingResult.Summary,
	}, nil
}
```

補足:
- `FetchResult` は `CreatedEmailIDs` / `CreatedEmails` / `ExistingEmailIDs` を持ち、workflow は通常 `CreatedEmails` を次段へ渡す。

## 6. `manualmailworkflow` で持つ port

### `internal/manualmailworkflow/application`
```go
type FetchStage interface {
	Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

type AnalyzeStage interface {
	Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

type BillingStage interface {
	Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}
```

### 対応する具体型
- `DirectManualMailFetchAdapter`
  - `mailfetch.UseCase` を呼ぶ
- `DirectMailAnalysisAdapter`
  - `mailanalysis.UseCase` を呼ぶ
- `DirectBillingAdapter`
  - `billing.UseCase` を呼ぶ

補足:
- 初期は direct adapter で十分
- 将来 queue 化したい場合は `QueuedMailAnalysisAdapter` や `QueuedBillingAdapter` に差し替え可能

## 7. Factory 設計

### 採用方針
- `Factory` は採用する
- 使いどころは provider / 実装を実行時に選ぶ点に限定する

### `MailFetcherFactory`
- 配置
  - `internal/mailfetch/application`
- 役割
  - `MailAccountConnection` から provider 種別を見て、適切な `MailFetcher` を返す
- 返すもの
  - `gmail` の場合は Gmail adapter
  - 将来 `outlook` が追加されたら Outlook adapter

```go
type MailFetcherFactory interface {
	Create(ctx context.Context, conn ConnectionRef) (MailFetcher, error)
}

type MailFetcher interface {
	Fetch(ctx context.Context, cond FetchCondition) ([]common.FetchedEmailDTO, []domain.MessageFailure, error)
}
```

### `AnalyzerFactory`
- 配置
  - `internal/mailanalysis/application`
- 役割
  - AI provider / model / tenant 設定に応じて適切な `Analyzer` を返す

```go
type AnalyzerFactory interface {
	Create(ctx context.Context, spec AnalyzerSpec) (Analyzer, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, email EmailForAnalysisTarget) (AnalysisOutput, error)
}
```

### `manualmailworkflow` で factory を使うケース
- 初期実装では不要
- 使うとすれば以下のケース
  - direct 実行と queued 実行を切り替える
  - 環境ごとに stage adapter を切り替える

補足:
- まずは DI で `FetchStage` / `AnalyzeStage` / `BillingStage` を直接差し込む方が単純
- `manualmailworkflow` 用の factory は必要になるまで作らない

### Factory を使わない箇所
- `BillingRepository`
  - 実装選択が発生しないなら factory は不要
- 固定の Gorm repository
  - 実装が 1 つなら factory は不要
- ドメインエンティティ生成
  - Go では constructor function を factory method として扱えば十分
  - 例
    - `NewEmailFromFetchedDTO`
    - `NewBilling`

## 8. Adapter 設計

### 採用方針
- `Adapter` は採用する
- 外部サービス、DB、他 package 呼び出しの境界に置く

### Adapter を置く境界
- 外部メールサービス
- AI provider
- DB 永続化
- workflow package から各 stage への呼び出し
- vendor 解決の外部参照

### Adapter 一覧
| package | port | adapter | 役割 |
| --- | --- | --- | --- |
| `manualmailworkflow` | `FetchStage` | `DirectManualMailFetchAdapter` | `mailfetch.UseCase` を呼ぶ |
| `manualmailworkflow` | `AnalyzeStage` | `DirectMailAnalysisAdapter` | `mailanalysis.UseCase` を呼ぶ |
| `manualmailworkflow` | `BillingStage` | `DirectBillingAdapter` | `billing.UseCase` を呼ぶ |
| `mailfetch` | `MailFetcher` | `GmailMailFetcherAdapter` | Gmail API から生メールを取得する |
| `mailfetch` | `ConnectionRepository` | `GormConnectionRepositoryAdapter` | connection と credential を読む |
| `mailfetch` | `EmailRepository` | `GormEmailRepositoryAdapter` | `Email` を batch 保存する |
| `mailanalysis` | `Analyzer` | `OpenAIAnalyzerAdapter` | AI 解析して `AnalysisOutput` を返す |
| `mailanalysis` | `EmailRepository` | `GormEmailReaderAdapter` | 解析対象 `Email` を読む |
| `mailanalysis` | `ParsedEmailRepository` | `GormParsedEmailRepositoryAdapter` | `ParsedEmail` を保存する |
| `billing` | `ParsedEmailRepository` | `GormParsedEmailReaderAdapter` | `ParsedEmail` を読む |
| `billing` | `EmailRepository` | `GormBillingSourceEmailReaderAdapter` | 参照元 `Email` を読む |
| `billing` | `VendorResolver` | `VendorResolverAdapter` | canonical Vendor を解決する |
| `billing` | `BillingRepository` | `GormBillingRepositoryAdapter` | 重複確認と `Billing` 保存を行う |

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
  - billing 起動

#### `DirectBillingAdapter`
- やること
  - `billing.UseCase` を呼ぶ
  - workflow 用の入出力に変換する
- やらないこと
  - billing ルール実装

#### `GmailMailFetcherAdapter`
- やること
  - token を使って Gmail API を叩く
  - `FetchCondition` に基づき対象メールを取得する
  - 生データを `FetchedEmailDTO` に変換する
- やらないこと
  - `Email` 保存
  - AI 解析
  - 請求判定

#### `OpenAIAnalyzerAdapter`
- やること
  - prompt を受けて OpenAI API を呼ぶ
  - 応答を構造化して draft を返す
- やらないこと
  - `ParsedEmail` 保存
  - vendor 解決
  - billing 生成

#### `VendorResolverAdapter`
- やること
  - alias テーブル、既知マッピング、送信元ルールなどを使って canonical Vendor を返す
- やらないこと
  - `BillingEligibility` 判定
  - `Billing` 保存

## 9. Abstract の扱い

### 結論
- abstract class は使わない

### 理由
- Go は継承ベースより interface + composition の方が自然
- 共通処理を持つ親クラスを作ると責務が曖昧になりやすい
- adapter ごとの差分より共通化の都合が優先されると、境界が崩れやすい

### 代替
- 小さい interface
  - `FetchStage`
  - `AnalyzeStage`
  - `BillingStage`
  - `MailFetcher`
  - `Analyzer`
  - `VendorResolver`
- constructor function
  - `NewBilling`
  - `NewEmailFromFetchedDTO`
- helper / support struct
  - retry helper
  - prompt builder
  - logging helper
- decorator
  - `LoggingFetchStage`
  - `RetryingAnalyzer`
  - `MetricsBillingRepository`

```go
type LoggingFetchStage struct {
	next FetchStage
	log  logger.Interface
}
```

## 10. 推奨 port

### `internal/manualmailworkflow/application`
```go
type FetchStage interface {
	Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

type AnalyzeStage interface {
	Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

type BillingStage interface {
	Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}
```

### `internal/mailfetch/application`
```go
type ConnectionRepository interface {
	FindUsableConnection(ctx context.Context, userID, connectionID uint) (ConnectionRef, error)
}

type MailFetcherFactory interface {
	Create(ctx context.Context, conn ConnectionRef) (MailFetcher, error)
}

type MailFetcher interface {
	Fetch(ctx context.Context, cond FetchCondition) ([]common.FetchedEmailDTO, []domain.MessageFailure, error)
}

type EmailRepository interface {
	SaveAllIfAbsent(ctx context.Context, userID uint, source domain.EmailSource, msgs []common.FetchedEmailDTO) ([]domain.SaveResult, []domain.MessageFailure, error)
}
```

### `internal/mailanalysis/application`
```go
type AnalyzerFactory interface {
	Create(ctx context.Context, spec AnalyzerSpec) (Analyzer, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, email EmailForAnalysisTarget) (AnalysisOutput, error)
}

type ParsedEmailRepository interface {
	SaveAll(ctx context.Context, input SaveInput) ([]ParsedEmailRecord, error)
}
```

### `internal/billing/application`
```go
type ParsedEmailRepository interface {
	FindForDerivation(ctx context.Context, parsedEmailID uint) (ParsedEmailForBilling, error)
}

type EmailRepository interface {
	FindSource(ctx context.Context, emailID uint) (SourceEmail, error)
}

type VendorResolver interface {
	Resolve(ctx context.Context, input VendorResolutionInput) (VendorResolution, error)
}

type BillingRepository interface {
	ExistsByIdentity(ctx context.Context, userID, vendorID uint, billingNumber string) (bool, error)
	Save(ctx context.Context, billing Billing) error
}
```

## 11. DI 方針

### 原則
- port interface は application に置く
- adapter 実装は infrastructure に置く
- DI は `internal/di` で行う
- presentation は `manualmailworkflow.UseCase` を直接呼ぶ

### 例
- `ProvideMailFetchDependencies`
  - `GormConnectionRepositoryAdapter`
  - `GormEmailRepositoryAdapter`
  - `DefaultMailFetcherFactory`
  - `mailfetch/application.UseCase`

- `ProvideMailAnalysisDependencies`
  - `GormEmailReaderAdapter`
  - `GormParsedEmailRepositoryAdapter`
  - `DefaultAnalyzerFactory`
  - `mailanalysis/application.UseCase`

- `ProvideBillingDependencies`
  - `GormParsedEmailReaderAdapter`
  - `GormBillingSourceEmailReaderAdapter`
  - `VendorResolverAdapter`
  - `GormBillingRepositoryAdapter`
  - `billing/application.UseCase`

- `ProvideManualMailWorkflowDependencies`
  - `DirectManualMailFetchAdapter`
  - `DirectMailAnalysisAdapter`
  - `DirectBillingAdapter`
  - `manualmailworkflow/application.UseCase`

## 12. 同期 / 非同期方針
- presentation は `manualmailworkflow` の開始だけを受ける。
- 推奨レスポンスは `202 Accepted`。
- 初期は `manualmailworkflow` から direct adapter で各 stage を同期的に呼んでもよい。
- 長時間化する場合は `manualmailworkflow` から queue adapter に差し替える。
- その場合でも全体フローの定義は `manualmailworkflow` に残し、各 stage の責務は変えない。
- DB transaction は各 stage 内に閉じ、外部 I/O をまたぐ workflow 全体 transaction は採用しない。

## 13. 今回の判断
- `ManualMailFetch` は Email 保存までで止める。
- AI 解析は `mailanalysis` に分離する。
- `VendorResolution` と `BillingEligibility` は `billing` に置く。
- `ParsedEmail` は `Billing` と分けて保存する。
- `internal/manualmailworkflow` を追加し、全体フローはそこで持つ。
- `mailfetch` / `mailanalysis` / `billing` は各段の usecase に責務を絞る。
- adapter は各段の実装を差し込む。
- factory は provider 切替点でのみ採用する。
- abstract class は使わない。
