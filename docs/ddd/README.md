# ドメインモデル

本ドキュメントは、billingrse のドメインモデルを概念レベルでまとめたドキュメントである。
集約境界、主要なエンティティ / 値オブジェクト / ポリシー、関係と依存関係を整理する。

注記:
- ここでは将来の到達像を含む概念モデルも扱う。
- 現行実装の `manualmailworkflow` は `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` までが接続済みである。

参照:
- `docs/ddd/ubiquitous-language/README.md`
- `docs/ddd/invariants.md`

## クラス図

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
  namespace 集約_バッチ設定 {
    class BatchSetting["バッチ設定（BatchSetting）"]
    class FetchCondition["取得条件（FetchCondition）"]
  }
  namespace 集約_メールアカウント連携 {
    class MailAccountConnection["メールアカウント連携（MailAccountConnection）"] {
      + 認可する()
      + 再認可する()
      + 連携解除する()
    }
  }
  namespace 集約_手動履歴 {
    class ManualMailWorkflowHistory["手動履歴（ManualMailWorkflowHistory）"]
    class ManualMailWorkflowFailure["手動履歴失敗（ManualMailWorkflowFailure）"]
  }
  namespace 集約_メール {
    class Email["メール（Email）"]
    class ParsedEmail["メール解析結果（ParsedEmail）"]
  }
  namespace ポリシー_支払先解決 {
    class VendorResolution["支払先解決（VendorResolution）"] {
      + 解決する()
    }
  }
  namespace ポリシー_請求成立判定 {
    class BillingEligibility["請求成立判定（BillingEligibility）"] {
      + 判定する()
    }
  }
  namespace 集約_請求 {
    class Billing["請求（Billing）"]
    class BillingLineItem["請求明細（BillingLineItem）"]
    class PaymentCycle["支払周期（PaymentCycle）"]
    class Money["金額（Money）"]
    class BillingNumber["請求番号（BillingNumber）"]
    class InvoiceNumber["インボイス番号（InvoiceNumber）"]
  }
  namespace 集約_支払先 {
    class Vendor["支払先（Vendor）"]
  }

  <<policy>> VendorResolution
  <<policy>> BillingEligibility
  <<enumeration>> PaymentCycle
  <<value_object>> FetchCondition
  <<value_object>> Money
  <<value_object>> BillingNumber
  <<value_object>> InvoiceNumber

  User "1" --> "0..*" Email : 所有
  Email "1" *-- "0..*" ParsedEmail : 解析結果
  ParsedEmail --> VendorResolution : 支払先候補
  VendorResolution <.. BillingEligibility : 解決結果を利用
  BillingEligibility ..> Billing : 成立対象を決める

  User "1" --> "0..*" ManualMailWorkflowHistory : 所有
  User "1" --> "0..*" MailAccountConnection : 連携
  User "1" --> "0..*" EmailVerificationToken : 認証
  User "1" --> "0..*" BatchSetting : 所有
  User "1" --> "0..*" Billing : 所有

  MailAccountConnection "0..*" --> "1" MailService : サービス
  BatchSetting *-- FetchCondition : 取得条件
  ManualMailWorkflowHistory *-- "0..*" ManualMailWorkflowFailure : failure

  ParsedEmail --> BillingEligibility : 成立判定
  VendorResolution --> Vendor : 解決

  Billing --> Vendor : 支払先
  Billing --> PaymentCycle : 支払周期
  Billing *-- Money : 請求総額
  Billing *-- "1..*" BillingLineItem : 請求明細
  BillingLineItem *-- Money : 明細金額
  Billing *-- BillingNumber : 請求番号
  Billing *-- InvoiceNumber : インボイス番号
  Billing ..> Email : 参照元
```

## 集約境界（概念レベル）

本システムで扱う集約ルートと、どの概念を内包または参照するかを整理する。

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
- 参照: ユーザー / メールサービス

### バッチ設定集約
- ルート: バッチ設定（BatchSetting）
- 含む: 取得条件（FetchCondition）
- 所有: ユーザー（User）
- 参照: メールアカウント連携（MailAccountConnection）
- 説明: ユーザーが所有するメール取得設定の集約

### 手動履歴集約
- ルート: 手動履歴（ManualMailWorkflowHistory）
- 含む: 手動履歴失敗（ManualMailWorkflowFailure）
- 所有: ユーザー（User）
- 説明: 1回の手動メール取得 workflow の受付条件、進行状態、stage 集計、failure を保持する監査用集約

### メール集約
- ルート: メール（Email）
- 含む: メール解析結果（ParsedEmail）
- 参照: ユーザー

### 請求集約
- ルート: 請求（Billing）
- 参照: 支払先（Vendor）/ メール（Email）/ 支払周期（PaymentCycle）
- 含む（エンティティ）: 請求明細（BillingLineItem）
- 含む（値オブジェクト）: 請求総額（Money）/ 明細金額（Money）/ 請求番号（BillingNumber）/ インボイス番号（InvoiceNumber）

### 支払先集約
- ルート: 支払先（Vendor）
- 説明: 参照データの集約

## 概念モデル

詳細な属性設計や実装方針は別途策定する。

### エンティティ

#### ユーザー（User）
- サービスを利用する主体
- データ分離の単位
- ユーザー名・メールアドレスを持つ
- パスワードハッシュを保持する
- メール認証状態と認証日時を持つ

#### メール認証トークン（EmailVerificationToken）
- ユーザーのメール認証に使うトークン
- 有効期限と消費状態を持つ

#### メールサービス（MailService）
- Gmail / Outlook などのサービス種別

#### メールアカウント連携（MailAccountConnection）
- ユーザーの外部メールサービス接続情報
- 認可情報（アクセストークン / リフレッシュトークン）を持つ

#### バッチ設定（BatchSetting）
- ユーザーが所有するメール取得設定の集約ルート
- 取得条件（FetchCondition）と実行スケジュールを持つ

#### 手動履歴（ManualMailWorkflowHistory）
- 手動メール取得 workflow 1回分の受付・進行・結果サマリを表す集約ルート
- 受付時点の手動メール取得条件と実行結果サマリを保持する
- stage ごとの成功件数 / 失敗件数と failure を保持する
- workflow 全体を失敗させた top-level error message を親に任意で保持できる
- `Email` / `ParsedEmail` / `Billing` の正本を置き換えない

#### 手動履歴失敗（ManualMailWorkflowFailure）
- 手動履歴にぶら下がる stage 単位の failure 記録
- `stage`、`reason_code`、ユーザー表示用 `message`、必要に応じて `external_message_id` を持つ
- append-only の明細として保持する
- 単独の集約にせず、必ず手動履歴配下で扱う

#### メール（Email）
- 取得したメールを表す
- Billing の参照元として保持される

#### メール解析結果（ParsedEmail）
- Email から生成される推定データ
- Email に紐づく解析履歴として保持される
- 解析結果の各フィールドは推定値のため、厳密な値オブジェクトは持たせない（プリミティブで保持）

#### 支払先（Vendor）
- 正規化された事業者・サービス
- ParsedEmail 上の候補名とは別の canonical な概念

#### 請求（Billing）
- 金額（Money）・支払先・請求番号（BillingNumber）・インボイス番号（InvoiceNumber）などが確定した支払いの事実
- 1件以上の請求明細（BillingLineItem）を内包する集約ルート
- `Billing.Money` は請求総額を表し、正本は請求明細の内訳とする

#### 請求明細（BillingLineItem）
- 1つの請求にぶら下がる明細行
- 商品名・金額・通貨などの内訳を表す
- 単独の集約にせず、Billing 配下で扱う
- `BillingLineItem.Money` を正本として扱い、Billing の総額はその集約結果として扱う

### 概念 / ポリシー / 列挙

#### 請求成立判定（BillingEligibility）
- ParsedEmail を入力として成立可否を判断するポリシー
- 永続化されない
- Vendor の正規化そのものは担わず、支払先解決の結果を利用する

#### 支払先解決（VendorResolution）
- ParsedEmail の支払先候補やメール由来の情報をもとに、正規化済み Vendor を決定するポリシー
- BillingEligibility とは別責務である
- 表記揺れや AI 解析結果の揺らぎを吸収する

#### 支払周期（PaymentCycle）
- 請求が単発か定期かを表す分類

#### メール認証（EmailVerification）
- EmailVerificationToken を用いてメールアドレスの正当性を確認する手続き
- 永続化されない

### 値オブジェクト（各集約に内包）

#### 取得条件（FetchCondition）
- メール取得の対象期間とラベルの組
- BatchSetting 集約に内包される
- 期間とラベルは必須

#### 金額（Money）
- 金額と通貨の組
- 金額は小数第3位までを許容する
- 通貨は ISO 4217 の3文字コード
- 当面は JPY / USD のみに限定する

#### 請求番号（BillingNumber）
- ベンダーが発行する請求書の識別子
- 必須

#### インボイス番号（InvoiceNumber）
- 適格請求書発行事業者登録番号
- 形式は "T" + 数字13桁
- 任意（存在しない請求もある）

## 主要な関係（概念）

- ユーザーは複数のメールアカウント連携を持つ
- ユーザーは複数のメール認証トークンを持つ
- ユーザーは複数のバッチ設定を持つ
- ユーザーは複数の手動履歴を持つ
- メールアカウント連携は1つのメールサービスに紐づく
- BatchSetting は FetchCondition を持つ
- 手動履歴は受付時点の手動メール取得条件を持つ
- 手動履歴は複数の手動履歴失敗を内包する
- Email は ParsedEmail を保持する
- ParsedEmail は支払先解決の入力になる
- BillingEligibility は ParsedEmail と支払先解決の結果を用いて成立判定する
- Billing は Vendor を参照し、参照元として Email を持つ
- Billing は 1件以上の BillingLineItem を内包する
- Billing の総額は BillingLineItem の金額内訳を正本として扱う

## 依存関係（モデル）

- User / MailService / Vendor / PaymentCycle / FetchCondition / Money / BillingNumber / InvoiceNumber は他に依存しない
- MailAccountConnection -> User, MailService
- BatchSetting -> User, FetchCondition
- ManualMailWorkflowHistory -> User, ManualMailWorkflowFailure
- Email -> User
- ParsedEmail -> Email
- VendorResolution（ポリシー） -> ParsedEmail, Vendor
- BillingEligibility（ポリシー） -> ParsedEmail, VendorResolution の結果
- Billing -> Vendor, PaymentCycle, Money, BillingNumber, InvoiceNumber, BillingLineItem, Email（参照元）
- BillingLineItem -> Money

## 請求生成フロー（シーケンス図）

`VendorResolution` を含む、Email 取得から Billing 作成までの代表的な流れを示す。

```mermaid
sequenceDiagram
  actor Trigger as 手動取得 / バッチ
  participant App as ApplicationService
  participant Email as Email
  participant AI as AI解析
  participant Parsed as ParsedEmail
  participant Resolve as VendorResolution
  participant Eligibility as BillingEligibility
  participant Billing as Billing

  Trigger->>App: メール取得を実行
  App->>Email: Email を保存
  App->>AI: Email を解析
  AI-->>App: 推定結果を返す
  App->>Parsed: ParsedEmail を保存
  Note over Parsed,Resolve: ParsedEmail の支払先名は候補値であり canonical Vendor ではない
  App->>Resolve: ParsedEmail とメール由来情報で Vendor を解決

  alt Vendor を解決できた
    Resolve-->>App: canonical Vendor を返す
    App->>Eligibility: ParsedEmail + VendorResolution の結果を渡す

    alt 請求成立条件を満たす
      Eligibility-->>App: eligible
      App->>Billing: Billing を作成
      Note over Billing: 参照元として保持するのは Email
      Billing-->>App: 保存完了
    else 請求不成立
      Eligibility-->>App: not eligible
      App-->>Trigger: Billing は作成しない
    end

  else Vendor が未解決
    Resolve-->>App: unresolved
    App-->>Trigger: Billing は作成しない（未解決件数としてカウントし、technical failure とは分離する）
  end
```

## 補足

- 不変条件の詳細は `docs/ddd/invariants.md` を正とする
