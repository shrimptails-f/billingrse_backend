# ユビキタス言語

- 以下は設計・実装・会話で共通に使う言葉
- 事実／判断／仮説が混ざらないように整理済み


# Ubiquitous Language Class Diagram

```mermaid
classDiagram
  class User["ユーザー（User）"]
  class MailService["メールサービス（MailService）"]
  class MailAccountConnection["メールアカウント連携（MailAccountConnection）"]
  class MailFetch["メール取得（MailFetch）"]
  class ManualMailFetch["手動メール取得（ManualMailFetch）"]
  class MailFetchBatch["メール取得バッチ（MailFetchBatch）"]
  class BatchSetting["バッチ設定（BatchSetting）"]
  class FetchCondition["取得条件（FetchCondition）"]
  class Email["メール（Email）"]
  class ParsedEmail["メール解析結果（ParsedEmail）"]
  class BillingEligibility["請求成立判定（BillingEligibility）"]
  class Billing["請求（Billing）"]
  class Vendor["支払先（Vendor）"]
  class PaymentCycle["支払周期（PaymentCycle）"]
  class Money["金額（Money）"]
  class BillingNumber["請求番号（BillingNumber）"]
  class InvoiceNumber["インボイス番号（InvoiceNumber）"]

  class User {
    + ログインする()
    + ログアウトする()
  }

  class MailAccountConnection {
    + 認可する()
    + 再認可する()
    + 失効する()
  }

  class ManualMailFetch {
    + 実行する()
  }

  class MailFetchBatch {
    + 実行する()
  }

  class BillingEligibility {
    + 判定する()
  }

  <<concept>> MailFetch
  <<policy>> BillingEligibility
  <<enumeration>> PaymentCycle
  <<value_object>> Money
  <<value_object>> BillingNumber
  <<value_object>> InvoiceNumber

  User "1" --> "0..*" MailAccountConnection : 連携
  User "1" --> "0..*" BatchSetting : 所有
  User "1" --> "0..*" Email : 所有
  User "1" --> "0..*" Billing : 所有

  MailAccountConnection "0..*" --> "1" MailService : サービス
  MailAccountConnection "1" --> "0..*" MailFetch : 取得に使用

  MailFetch "1" o-- "0..*" ManualMailFetch : 手動
  MailFetch "1" o-- "0..*" MailFetchBatch : バッチ

  MailFetchBatch "1" --> "1" BatchSetting : 設定
  BatchSetting "1" --> "1" FetchCondition : 取得条件

  ManualMailFetch --> Email : 取得
  MailFetchBatch --> Email : 取得

  Email "1" --> "0..*" ParsedEmail : 解析結果
  ParsedEmail --> BillingEligibility : 成立判定
  BillingEligibility --> Billing : 生成

  Billing --> Vendor : 支払先
  Billing --> PaymentCycle : 支払周期
  Billing *-- Money : 金額
  Billing *-- BillingNumber : 請求番号
  Billing *-- InvoiceNumber : インボイス番号
  Billing ..> Email : 参照元
  Billing ..> ParsedEmail : 参照元
```

## カテゴリ別ファイル

| カテゴリ名 | 言語 |
| --- | --- |
| [ユーザー系](user.md) | ユーザー（User）,ログイン（Login）,ログアウト（Logout）,ユーザー名（UserName）,メールアドレス（EmailAddress）,パスワード（Password）,パスワードハッシュ（PasswordHash）,メール認証（EmailVerification）,メール認証トークン（EmailVerificationToken） |
| [メール連携/取得系](mail-integration-fetch.md) | メールサービス（MailService）,メールアカウント連携（MailAccountConnection）,メール取得（MailFetch）,手動メール取得（ManualMailFetch）,メール取得バッチ（MailFetchBatch）,バッチ設定（BatchSetting）,取得条件（FetchCondition） |
| [メール/解析系](mail-analysis.md) | メール（Email）,メール解析結果（ParsedEmail）,請求成立判定（BillingEligibility） |
| [請求/支払先系](billing-vendor.md) | 支払先（Vendor）,請求（Billing）,支払周期（PaymentCycle）,金額（Money）,請求番号（BillingNumber）,インボイス番号（InvoiceNumber） |

## 用語間の関係（言語レベル）

```
ユーザー
  ↓ 連携
メールアカウント連携
  ↓
メール取得（手動メール取得/メール取得バッチ）
  ↓
メール
  ↓ 解析
メール解析結果
  ↓ 請求成立判定
請求
  └─ 支払先

メールサービス
  ↓ 連携
メールアカウント連携
  ↓
メール取得（手動メール取得/メール取得バッチ）
  ↓
メール
  ↓ 解析
メール解析結果
  ↓ 請求成立判定
請求
  └─ 支払先
```

## 意図的に未定義としている言葉

注記: 今は使わない（将来のサブドメイン）

- 支出
- 家計簿
- 会計
- 月次集計
- 合計金額
