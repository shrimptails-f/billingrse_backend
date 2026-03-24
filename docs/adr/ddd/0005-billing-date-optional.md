# ADR 0005: billing_date を任意項目として扱う

Status: Accepted  
Date: 2026-03-25

## Context
ADR 0003 では、請求成立の必須項目に「請求日」を含めていた。
しかし実際の請求メールには、請求日が明示されていないものが存在する。

現行の `ParsedEmail` は `billingDate` を optional として保持でき、
`BillingEligibility` も `billing_date == nil` を許容する方向で整理されている。
一方で `Billing` aggregate 側で請求日を必須にすると、次の問題が起きる。

- メール上に請求日が存在しないだけで、成立可能な請求を保存できなくなる
- `billingeligibility` で eligible でも、後続 `billing` stage で生成失敗する
- `received_at` など別の日時を自動代入すると、原文に存在しない事実をドメイン値として確定してしまう

このため、「請求日が存在しないメールがある」という現実を優先し、
請求日の欠落をドメイン上の不成立条件にはしない方針を明確にする必要があった。

## Decision
- `billing_date` は請求生成における任意項目とする
- `BillingEligibility` は `billing_date == nil` を理由に non-eligible としない
- `Billing` aggregate は `billing_date` を `nil` で保持できるようにする
- `billing_date` が取得できた場合のみ、その値を正規化して保持する
- 請求の同一性は引き続き「ユーザー + Vendor + 請求番号」で判定し、`billing_date` は同一性に使わない
- 原文に存在しない請求日を、`received_at` や処理日時で自動補完しない

## Consequences
- ADR 0003 の「請求成立の必須項目は Vendor + 金額 + 請求日 + 支払周期 + 請求番号」とした判断のうち、
  請求日必須の部分は本 ADR で上書きされる
- DDD 文書、不変条件、`Billing` aggregate、後続 `billing` stage は `billing_date` が `nil` のケースを扱う必要がある
- API / UI / 集計処理では、請求日が未設定の請求を明示的に扱う必要がある
- 将来、請求日が業務上必須になった場合は、自動補完ではなく手動補完や別ワークフロー導入を別途設計する
