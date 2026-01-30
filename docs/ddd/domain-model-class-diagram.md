# ドメインモデル クラス図

```mermaid
classDiagram
  namespace 集約_ユーザー {
    class User["ユーザー（User）"] {
      + ログインする()
      + ログアウトする()
    }
    class EmailVerificationToken["メール認証トークン（EmailVerificationToken）"]
  }
  namespace 集約_メールサービス {
    class MailService["メールサービス（MailService）"]
  }
  namespace 集約_メールアカウント連携 {
    class MailAccountConnection["メールアカウント連携（MailAccountConnection）"] {
      + 認可する()
      + 再認可する()
      + 失効する()
    }
  }
  namespace 集約_バッチ設定 {
    class BatchSetting["バッチ設定（BatchSetting）"] {
      + 取得条件（FetchCondition）
    }
  }
  namespace 集約_メール {
    class Email["メール（Email）"]
    class ParsedEmail["メール解析結果（ParsedEmail）"]
  }
  namespace ポリシー_請求成立判定 {
    class BillingEligibility["請求成立判定（BillingEligibility）"] {
      + 判定する()
    }
  }
  namespace 集約_請求 {
    class Billing["請求（Billing）"]
    class PaymentCycle["支払周期（PaymentCycle）"]
    class Money["金額（Money）"]
    class BillingNumber["請求番号（BillingNumber）"]
    class InvoiceNumber["インボイス番号（InvoiceNumber）"]
  }
  namespace 集約_支払先 {
    class Vendor["支払先（Vendor）"]
  }

  <<policy>> BillingEligibility
  <<enumeration>> PaymentCycle
  <<value_object>> Money
  <<value_object>> BillingNumber
  <<value_object>> InvoiceNumber

  User "1" --> "0..*" MailAccountConnection : 連携
  User "1" --> "0..*" EmailVerificationToken : 認証
  User "1" --> "0..*" BatchSetting : 所有
  User "1" --> "0..*" Email : 所有
  User "1" --> "0..*" Billing : 所有

  MailAccountConnection "0..*" --> "1" MailService : サービス

  Email "1" --> "0..*" ParsedEmail : 解析結果
  ParsedEmail --> BillingEligibility : 成立判定
  BillingEligibility --> Billing : 生成

  Billing --> Vendor : 支払先
  Billing --> PaymentCycle : 支払周期
  Billing *-- Money : 金額
  Billing *-- BillingNumber : 請求番号
  Billing *-- InvoiceNumber : インボイス番号
  Billing ..> Email : 参照元
```

# 集約境界（概念レベル）

本ドキュメントは、ドメインモデルの集約境界を概念レベルで整理する。

参照:
- `docs/ddd/ubiquitous-language/README.md`
- `docs/ddd/invariants.md`
- `docs/ddd/domain-model.md`

## 集約一覧

### ユーザー集約
- ルート: ユーザー（User）
- 含む: メール認証トークン（EmailVerificationToken）
- 説明: データ分離の単位

### メールサービス集約
- ルート: メールサービス（MailService）
- 説明: 参照データの集約

### メールアカウント連携集約
- ルート: メールアカウント連携（MailAccountConnection）
- バッチ設定: 取得条件 / 実行スケジュール（いずれも連携に紐づく）
- 参照: ユーザー / メールサービス

### メール集約
- ルート: メール（Email）
- 含む: メール解析結果（ParsedEmail）
- 参照: ユーザー

### 請求集約
- ルート: 請求（Billing）
- 参照: 支払先（Vendor）/ メール（Email）/ 支払周期（PaymentCycle）
- 含む（値オブジェクト）: 金額（Money）/ 請求番号（BillingNumber）/ インボイス番号（InvoiceNumber）

### 支払先集約
- ルート: 支払先（Vendor）
- 説明: 参照データの集約

## 補足

- 不変条件の詳細は `docs/ddd/invariants.md` を正とする
