# ADR 0003: 請求番号とインボイス番号の分離、支払周期の採用

Status: Accepted  
Date: 2026-01-30

## Context
請求番号（ベンダーが発行する番号）とインボイス制度における番号が混同されやすく、
InvoiceNumber の意味が曖昧だった。インボイス番号を持たない事業者も存在するため、
請求成立要件や識別子の扱いを見直す必要があった。また、支払いの性質よりも
単発/定期の区別が主要な要件となった。

## 経緯
- 当初は InvoiceNumber を「請求番号」として扱っていたが、インボイス制度の文脈と衝突した。
- 請求番号が存在しない事業者も想定されるため、請求番号を必須とする前提を見直した。
- 「性質（PaymentType）」よりも「単発/定期（PaymentCycle）」の区別が要件として重要だった。
- 上記を踏まえ、請求番号とインボイス番号を分離し、PaymentCycle を採用する方針に整理した。

## Decision
- 請求番号（BillingNumber）を新設し、ベンダーが発行する請求書識別子として扱う。
  - BillingNumber は任意（nil/空は許容）。
- InvoiceNumber は「適格請求書発行事業者登録番号（インボイス番号）」と定義する。
  - 形式は "T" + 数字13桁。
  - 任意（nil/空は許容）。
- 請求成立の必須項目は「Vendor + 金額」とする。
- 支払いの分類は PaymentType ではなく PaymentCycle（単発/定期）を採用する。
- ParsedEmail の推定項目は以下に整理する。
  - vendorName, billingNumber, invoiceNumber, amount, currency, billingDate, paymentCycle, extractedAt
  - amount は小数第3位までの数値を許容する。

## Consequences
- ADR 0001/0002 の「請求番号必須」「支払いタイプ」等の記述は、本ADRにより実質的に上書きされる。
- 解析プロンプトや DTO で billingNumber / invoiceNumber / paymentCycle の取り扱いが必要になる。
- InvoiceNumber の検証は "T" + 13桁に限定されるため、他形式は BillingNumber に格納する。
