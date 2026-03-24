# 請求/支払先系

## 支払先（Vendor）

### 定義
- 請求の支払い対象となる、正規化された事業者・サービス

### 例
- Netflix
- AWS
- Google

### ルール
- 表記揺れは Vendor に集約する
- Vendor は請求より長寿命の概念
- Vendor は ParsedEmail の生文字列から直接同一視しない

## 支払先解決（VendorResolution）

### 定義
- ParsedEmail の支払先候補やメール由来の情報から、正規化済み Vendor を決定する処理

### 説明
- 表記揺れや AI 解析結果の揺らぎを吸収する
- 送信元や既知の別名などを使って canonical な Vendor に寄せる

### ルール
- BillingEligibility とは別責務である
- BillingEligibility は VendorResolution の結果を利用する
- Vendor が未解決のまま Billing を確定しない

## 請求（Billing）

### 定義（確定）
- 金額・日付・支払先が確定した、支払いの事実

### 必須属性
- 請求ID
- ユーザーID
- 請求番号(BillingNumber)
- 支払先（Vendor）
- 金額（Money）
- 請求日
- 支払周期（単発 / 定期）
- 参照元（Email）
### 任意属性
- インボイス番号（適格請求書発行事業者登録番号）

### ルール
- 請求は必ず Vendor を持つ
- 請求は必ず Email を参照元として持つ
- 請求は「支払われたかどうか」を表さない
- 請求の同一性は「ユーザー + Vendor + 請求番号」で判定する
- Billing に設定される Vendor は VendorResolution 済みの canonical な Vendor である
- ParsedEmail は請求成立判定の根拠として保存されるが、Billing の参照元属性には含めない
- 金額 / 請求番号 / インボイス番号は Billing 集約に内包される値オブジェクト

## 支払周期（PaymentCycle）

### 定義
- 請求が単発か定期かを表す分類

### 説明
- 定期
- 単発

### 備考
- 会計的な意味付けは行わない

## 金額（Money）

### 定義
- 金額と通貨の組

### 形式
- 金額は小数第3位までを許容
- 通貨は ISO 4217 の3文字コード
- 初期スコープでは JPY / USD のみを許容する

## 請求番号（BillingNumber）

### 定義
- ベンダーが発行する請求書の識別子

### 説明
- 英数字や記号を含む場合がある
- 必須（空は不可）

## インボイス番号（InvoiceNumber）

### 定義
- 適格請求書発行事業者登録番号

### 形式
- "T" + 数字13桁
