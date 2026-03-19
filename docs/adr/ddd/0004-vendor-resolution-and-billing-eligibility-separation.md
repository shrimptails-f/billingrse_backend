# ADR 0004: VendorResolution の分離と BillingEligibility の責務明確化

Status: Accepted  
Date: 2026-03-19

## Context
AI によるメール解析結果は毎回完全に安定した値になるとは限らず、支払先名には表記揺れや
推定の揺らぎが生じる。`ParsedEmail.vendorName` をそのまま canonical な `Vendor` と見なして
`BillingEligibility` の中で請求成立判定まで行うと、以下の問題が起きる。

- 支払先の正規化責務と請求成立判定責務が混ざる
- alias 追加や手動補正による再解決がしづらい
- AI 出力の揺らぎがそのまま `Billing` 作成判定に影響する
- `Billing` の同一性である「ユーザー + Vendor + 請求番号」が不安定になる

既存のドメインでは `Billing` の参照元は `Email` であり、`ParsedEmail` は推定結果として扱っている。
この前提に合わせ、支払先の確定と請求成立判定を分離する必要があった。

## Decision
- `ParsedEmail.vendorName` は canonical な `Vendor` ではなく、支払先候補名として扱う。
- `VendorResolution` を独立したポリシー/ドメインモデルとして導入し、
  `ParsedEmail` とメール由来の情報から canonical な `Vendor` を解決する。
- `BillingEligibility` は `VendorResolution` の結果を入力として利用するが、
  `Vendor` の正規化そのものは担わない。
- `Vendor` が未解決のまま `Billing` を作成しない。
- `Billing` の同一性は引き続き「ユーザー + Vendor + 請求番号」とする。

## Consequences
- AI 解析の揺らぎは `VendorResolution` 側で吸収し、請求成立判定はより安定した入力で行える。
- alias 追加、既知 vendor への寄せ、手動補正などの改善ポイントを
  `BillingEligibility` から切り離して管理できる。
- ドメイン実装では、`BillingEligibility` は生の `vendorName` ではなく、
  解決済み `Vendor` を含む `VendorResolution` を受ける形に更新される。
- 未解決の `ParsedEmail` は `Billing` にならず、再解決または保留の対象として扱う必要がある。
