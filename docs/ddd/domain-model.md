# ドメインモデル（概念レベル）

本ドキュメントはユビキタス言語を概念レベルで構造化したモデルを示す。詳細な属性設計や実装方針は別途策定する。

参照:
- `docs/ddd/ubiquitous-language/README.md`
- `docs/ddd/invariants.md`
- `docs/ddd/ubiquitous-language-class-diagram.md`

## エンティティ

### ユーザー（User）
- サービスを利用する主体
- データ分離の単位
- ユーザー名・メールアドレスを持つ
- パスワードハッシュを保持する
- メール認証状態と認証日時を持つ

### メール認証トークン（EmailVerificationToken）
- ユーザーのメール認証に使うトークン
- 有効期限と消費状態を持つ

### メールサービス（MailService）
- Gmail / Outlook などのサービス種別

### メールアカウント連携（MailAccountConnection）
- ユーザーの外部メールサービス接続情報
- 認可情報を持つ

### メール（Email）
- 取得した加工前の一次情報
- 参照元として保持される

### メール解析結果（ParsedEmail）
- Email から生成される推定データ
- 解析結果の各フィールドは推定値のため、厳密な値オブジェクトは持たせない（プリミティブで保持）

### 支払先（Vendor）
- 正規化された事業者・サービス

### 請求（Billing）
- 金額・支払先・請求番号などが確定した支払いの事実

## 概念 / ポリシー / 列挙

### 請求成立判定（BillingEligibility）
- ParsedEmail を入力として成立可否を判断するポリシー
- 永続化されない

### 支払いタイプ（PaymentType）
- 請求の性質を表す分類

### メール認証（EmailVerification）
- EmailVerificationToken を用いてメールアドレスの正当性を確認する手続き
- 永続化されない

## 値オブジェクト（後で定義）

候補の整理は別途行う。現時点では枠のみを用意する。

## 主要な関係（概念）

- ユーザーは複数のメールアカウント連携を持つ
- ユーザーは複数のメール認証トークンを持つ
- メールアカウント連携は1つのメールサービスに紐づく
- Email は ParsedEmail を生成する
- ParsedEmail は請求成立判定を経て Billing を生成する
- Billing は Vendor を参照する

## 依存関係（モデル）

- User / MailService / Vendor / PaymentType は他に依存しない
- MailAccountConnection -> User, MailService
- Email -> MailAccountConnection
- ParsedEmail -> Email
- BillingEligibility（ポリシー） -> ParsedEmail
- Billing -> Vendor, PaymentType, Email / ParsedEmail（参照元）

## 補足

- 不変条件は `docs/ddd/invariants.md` を正とする
- 集約境界やVOの詳細は今後の設計で確定する
