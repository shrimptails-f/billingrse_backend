# VendorResolution による Vendor 正規化処理 要件定義

## 背景
- DDD と ADR では、`ParsedEmail.vendorName` は canonical な `Vendor` ではなく候補値として扱うこと、`VendorResolution` と `BillingEligibility` を分離することが既に合意されている。
- 実装上は `mailfetch`・`mailanalysis`・`vendorresolution` が存在し、`manualmailworkflow` も `fetch -> analysis -> vendorresolution` まで接続されている。
- 現行実装では `parsed_emails` に AI 解析結果は保存されるが、`emails` には本文は保存されず、件名・送信元・宛先・受信日時などのメタデータのみが保存される。
- まず必要なのは `Billing` 生成ではなく、保存済みデータから canonical `Vendor` を決定的に引き当てる処理である。

## 目的
- `mailanalysis` が保存した `ParsedEmail` を入力に、`VendorResolution` により canonical な `Vendor` を解決できるようにする。
- `VendorResolution` を `BillingEligibility` や `Billing` 生成から切り離し、単独の責務として成立させる。
- 今後 `BillingEligibility` や `Billing` を実装するときに、その前段として再利用できる vendor 正規化処理を用意する。

## 機能要件
- `VendorResolution` は `user_id` と、保存済み `ParsedEmail` 実体および source email の必要メタデータをまとめた入力を受けて処理できること。
- `VendorResolution` は `ParsedEmail.vendorName` を候補値として扱い、canonical `Vendor` と同一視しないこと。
- `VendorResolution` は少なくとも以下の保存済み情報を使って解決できること。
  - `ParsedEmail` の候補値
  - 元 `Email` の件名
  - 元 `Email` の送信元
  - 元 `Email` の宛先
- `VendorResolution` は保存済みデータだけで再実行可能であること。
  - `emails` に本文は保存されていないため、本文依存の解決を必須にしない。
- `VendorResolution` は外部 AI 再呼び出しではなく、決定的な解決処理として扱うこと。
- `VendorResolution` の結果は少なくとも「解決済み」「未解決」を区別できること。
- 結果として少なくとも以下を返却できること。
  - 解決済み件数
  - 未解決件数
  - 解決済み `parsed_email_id -> Vendor` 対応
  - 未解決だった `external_message_id` 一覧（重複なし）
  - どのルールで解決したかを示す `matched_by`
- 1 回の実行内で複数 `ParsedEmail` を処理でき、部分成功・部分失敗を扱えること。
- vendor 解決ルールは将来の alias 追加や既知マッピング拡張に耐えられること。
- unresolved の candidate vendor 名から、`Vendor` と `name_exact` alias を自動補完できること。
- 自動補完は `sender_domain` / `sender_name` / `subject_keyword` には適用しないこと。

## 非機能要件
- 責務配置は `mailanalysis` へ逆流させないこと。
- `mailanalysis` の責務は引き続き `ParsedEmail` 保存までに限定し、canonical Vendor 解決を持ち込まないこと。
- `VendorResolution` は同じ入力に対して同じ解決結果を返す、再現可能な処理であること。
- rerun 時に同じ `parsed_email_id` 群から同じ `Vendor` 解決結果が得られること。
- 保存済みメール本文が存在しない前提でも成立する設計にすること。
- ログやレスポンスにメール本文や秘匿情報を不要に露出しないこと。
- 監査や障害調査のため、少なくとも vendor 未解決と stage failure を識別できること。
- package 間依存は既存の Clean Architecture 方針に従い、application port 経由で接続すること。

## 制約事項
- 現行 `manualmailworkflow` は `fetch -> analysis -> vendorresolution` まで接続済みであり、`BillingEligibility` 以降は未実装である。
- vendor 永続化は `vendors` / `vendor_aliases` と repository 実装まで存在し、手動補正 UI / API は未実装である。
- `internal/common/domain` に DDD のモデルを集約する運用である。
- `emails` テーブルには本文が保存されないため、`VendorResolution` は本文なしでも動作しなければならない。

## 確定事項
- package は `internal/vendorresolution` とする。
- `VendorResolution` は本文を使わず、`ParsedEmail` と保存済み `Email` の具体値だけで判定する。
- canonical `Vendor` は `vendors` と `vendor_aliases` を分けて管理する。
- 初期の解決優先順位は以下で固定する。
  - alias 完全一致
  - sender domain
  - sender 表示名
  - subject 補助判定
  - unresolved
- `subject` 補助判定は最長一致優先とし、同長で複数 vendor に競合した場合は unresolved とする。
- vendor 未解決時の監査は初期は構造化ログのみとする。
- `vendor_aliases` は同じ `alias_type + normalized_value` を複数 vendor に持てる設計とする。
- `name_exact` / `sender_domain` / `sender_name` の alias 競合時は domain policy が `created_at DESC, id DESC` で 1 件選ぶ。
- 判定ルールと自動登録ルールは `internal/common/domain/vendor_resolution.go` に置く。
- `internal/vendorresolution/infrastructure` は判定材料の収集と保存だけを担当する repository とする。
- ユーザー単位の上書きルールは初期スコープに含めず、後続エンハンスとする。

## 成功条件
- 同一 vendor の表記揺れや送信元揺れが、同一 canonical `Vendor` に正規化される。
- `ParsedEmail` と `Email` だけで `Vendor` 解決を再実行できる。
- `VendorResolution` の結果を `BillingEligibility` の前段入力としてそのまま利用できる。
- 将来 alias や既知 vendor マッピングを増やしても、`mailanalysis` 側の責務を変更せず改善できる。

## 非スコープ
- `BillingEligibility`
- `Billing` 生成
- 重複確認
- `billings` テーブル
- vendor 手動補正 UI / API
- vendor alias 管理画面
- バッチ workflow への適用
- LLM による再解析や再推定
- vendor 解決結果そのものの永続化方式の最終確定
- ユーザー単位の vendor 上書きルール
