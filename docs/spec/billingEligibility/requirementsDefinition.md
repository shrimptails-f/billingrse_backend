# BillingEligibility stage 要件定義

## 背景
- DDD と ADR では、`BillingEligibility` は `VendorResolution` と分離された独立ポリシーとして扱うことが合意されている。
- 現行実装では `mailfetch`・`mailanalysis`・`vendorresolution` が概ね実装済みで、`manualmailworkflow` も `fetch -> analysis -> vendorresolution` まで接続済みである。
- `internal/common/domain` には `BillingEligibility` policy が既に存在し、請求成立に必要な必須項目チェックも定義されている。
- 一方で `internal/billingeligibility` stage と、その `manualmailworkflow` への接続は未実装である。
- メール本文由来の digest 生成と、`billing_number` 不在時の fallback 採番ルールは別タスクで扱い、ここでは「`billing_number` は受け取れる前提」とする。

## 目的
- `VendorResolution` で canonical `Vendor` が解決済み、かつ `billing_number` を受け取った対象に対し、請求成立可否を決定的に判定できるようにする。
- `BillingEligibility` を `Billing` 生成・重複確認・保存から切り離し、独立した stage / usecase として成立させる。
- 後続の `billing` stage が、そのまま `Billing` 生成に使える入力を受け取れるようにする。

## 機能要件
- `BillingEligibility` は `user_id` と、`VendorResolution` 済みの対象一覧を入力として処理できること。
- 各対象は少なくとも以下の情報を持っていること。
  - `parsed_email_id`
  - `email_id`
  - `external_message_id`
  - 解決済み `Vendor`
  - `matched_by`
  - `ParsedEmail` の判定必要項目
- 判定は少なくとも以下の成立条件を扱えること。
  - canonical `Vendor` が解決済みであること
  - `amount` が存在し、ドメイン上有効であること
  - `currency` が存在し、ドメイン上有効であること
  - `billing_number` が存在すること
  - `payment_cycle` が存在し、ドメイン上有効であること
- `billing_date` は任意項目として扱い、`nil` を許容すること。
- `invoice_number` は任意項目として扱い、存在すれば後続へ引き渡せること。
- 結果として少なくとも以下を返却できること。
  - 成立件数
  - 非成立件数
  - 成立した `eligible_items`
  - 非成立だった `ineligible_items`
  - stage failure 一覧
- `eligible_items` は後続の `billing` stage が `Billing` 生成に必要な情報を追加参照なしで利用できること。
- 1 回の実行で複数件を処理でき、部分成功・部分失敗を扱えること。
- `VendorResolution` 未解決は原則として前段で処理済みとし、`BillingEligibility` には解決済み対象のみが渡ること。
  - ただし不正な入力が渡された場合は failure として識別できること。

## 非機能要件
- `BillingEligibility` は外部 AI 再呼び出しや外部メールサービス呼び出しを行わないこと。
- 判定は同じ入力に対して同じ結果を返す、再現可能な処理であること。
- package 間依存は既存の Clean Architecture 方針に従い、application port 経由で接続すること。
- ログやレスポンスにメール本文や不要な秘匿情報を含めないこと。
- `mailanalysis` / `vendorresolution` の責務へ逆流しないこと。

## 制約事項
- 現行 `manualmailworkflow` は `analysis` 結果として `ParsedEmail` 実体を workflow payload に保持している。
- 現行 `vendorresolution` は workflow から `ParsedEmail` 実体を受け取り処理しているが、返却結果には判定用 `ParsedEmail` 全体を含めていない。
- `billingeligibility` stage は未実装であり、`manualmailworkflow` / DI / presentation の拡張が必要である。
- 後続の `billing` stage も未実装であるため、今回は保存処理ではなく判定と次段入力整備に責務を限定する。
- `billing_number` をどう生成するかは、このタスクの責務に含めない。
- `BillingEligibility` は `billing_number` を入力として受け取る前提に固定する。

## 確定事項
- package は `internal/billingeligibility` とする。
- 判定本体は既存の `internal/common/domain.BillingEligibility` をベースにしつつ、`billing_date` 任意化に合わせて見直す。
- `BillingEligibility` 自体は永続化しない。
- 非成立理由は Go の生 error 文字列ではなく、安定した reason code へ写像して扱う。
- 今回は `ParsedEmail` の再読込を前提にせず、workflow が保持している payload を活用する方向で設計する。
- `BillingEligibility` は `Vendor` 解決や `Billing` 保存を行わない。
- メール本文 digest の生成、`billing_number` 不在時の fallback 生成、`emails` テーブル保存は別設計に分離する。

## 成功条件
- `VendorResolution` 済み対象から、請求成立可否を決定的に再実行できる。
- 不足項目や不正値がある対象を reason code 付きで非成立として返せる。
- 成立対象を、そのまま `billing` stage の入力として引き渡せる。
- `vendorresolution` の責務を変更せず、`manualmailworkflow` に `billingeligibility` stage を追加できる。
- `billing_date == nil` の `ParsedEmail` も、他の成立条件を満たせば eligible にできる。
- `billing_number` がない入力は `BillingEligibility` の前段で補完済みとして扱える。

## 非スコープ
- `Billing` 生成
- 重複確認
- `billings` テーブル保存
- 請求成立判定結果の永続化
- vendor 手動補正 UI / API
- バッチ workflow への適用
