# Billing stage 要件定義

## 背景
- 現行実装では `mailfetch`・`mailanalysis`・`vendorresolution`・`billingeligibility`・`billing` が実装済みで、`manualmailworkflow` も `fetch -> analysis -> vendorresolution -> billingeligibility -> billing` まで接続済みである。
- `internal/common/domain` には `Billing`, `Money`, `BillingNumber`, `InvoiceNumber`, `PaymentCycle` が既に存在し、DDD 上の請求不変条件も整理済みである。
- DDD では請求の同一性を `user + vendor + billing_number` で判定し、`billing_date` は任意 (`nil` 許容) としている。
- 本ドキュメントでは、`billingeligibility` で成立した対象をどの契約で `Billing` に変換し、どう重複保存を防ぐかを明文化する。

## 目的
- `BillingEligibility` の `eligible_items` から `Billing` を生成し、重複を防ぎながら保存できるようにする。
- duplicate を top-level error にせず、業務結果として created / duplicate / failure を分けて返せるようにする。
- `manualmailworkflow` の `billing` stage を含めても、前段責務を壊さずに end-to-end の結果を返せるようにする。

## 機能要件
- `billing` stage は `user_id` と `eligible_items` 一覧を入力として処理できること。
- 各 `eligible_item` は少なくとも以下を含むこと。
  - `parsed_email_id`
  - `email_id`
  - `external_message_id`
  - `vendor_id`
  - `vendor_name`
  - `matched_by`
  - `billing_number`
  - `invoice_number`
  - `amount`
  - `currency`
  - `billing_date`
  - `payment_cycle`
- 各入力について `common/domain.NewBilling(...)` 相当のドメイン検証を通したうえで `Billing` を生成できること。
- 請求の同一性は `user + vendor + billing_number` で判定すること。
- duplicate は異常終了ではなく業務結果として返せること。
- 結果として少なくとも以下を返却できること。
  - `created_items`
  - `created_count`
  - `duplicate_items`
  - `duplicate_count`
  - `failures`
- `created_items` は少なくとも `billing_id` と入力元を対応付ける情報を持つこと。
- `duplicate_items` は少なくとも既存 `billing_id` と入力元を対応付ける情報を持つこと。
- 1 回の実行で複数件を処理でき、部分成功・部分失敗を扱えること。
- `eligible_items == 0` の場合は空結果を返し、保存処理を行わないこと。

## 非機能要件
- stage は外部メールサービスや AI を呼び出さないこと。
- 同じ `eligible_item` を再実行しても二重に `Billing` を作らず、同じ identity なら duplicate として扱えること。
- package 間依存は既存の Clean Architecture 方針に従い、application port 経由で接続すること。
- ログやレスポンスにメール本文や不要な秘匿情報を含めないこと。
- duplicate 判定はアプリケーションの事前確認だけに頼らず、DB 一意制約でも守られること。

## 制約事項
- 現行 workflow は `mailanalysis -> vendorresolution -> billingeligibility -> billing` を workflow payload で接続している。
- `billing` stage も原則として追加の `ParsedEmail` / `Email` 再読込を前提にせず、`eligible_items` だけで `Billing` 生成を完結できる形にする。
- `Billing` の参照元は `Email` であり、`ParsedEmail` を参照元として保持しない。
- `billing_date` は任意であり、メールに存在しないケースを許容する。
- duplicate 判定を `Exists + Save` の 2 段階に分けると競合時に race するため、保存契約は idempotent に設計する。

## 確定事項
- package は `internal/billing` とする。
- `Billing` aggregate 本体は `internal/common/domain` を再利用する。
- `billing` stage は `VendorResolution` や `BillingEligibility` を再実行しない。
- duplicate は top-level error ではなく stage result の一部として返す。
- 保存の最終防衛線として `user_id + vendor_id + billing_number` の DB 一意制約を置く。
- `billing` stage は `manualmailworkflow` / DI / presentation まで含めて接続する前提で設計する。

## 成功条件
- `eligible_items` から追加参照なしで `Billing` を生成・保存できる。
- 同じ入力を再実行しても duplicate として観測でき、二重登録されない。
- created / duplicate / failure が分離され、workflow result から controller response を組み立てられる。
- `billing_date == nil` の対象でも、他の要件を満たせば保存できる。
- `billing` の責務が「生成・重複制御・保存」に限定され、前段責務と混ざらない。

## 非スコープ
- `VendorResolution`
- `BillingEligibility`
- `billing_number` の生成
- `eligible_item` 補正 UI / API
- `Billing` 更新 / 削除
- バッチ workflow への適用
