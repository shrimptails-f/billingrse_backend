# VendorResolution による Vendor 正規化処理 要件定義

## 背景
- DDD と ADR では、`ParsedEmail.vendorName` は canonical な `Vendor` ではなく候補値として扱うこと、`VendorResolution` と `BillingEligibility` を分離することが既に合意されている。
- 実装上は `mailfetch` と `mailanalysis` は整備済みで、`manualmailworkflow` も `fetch -> analysis` までは実装されている。
- 一方で `billing` package は未実装であり、`VendorResolution` を使った canonical Vendor 解決と `Billing` 生成の段が未接続である。
- 現行実装では `parsed_emails` に AI 解析結果は保存されるが、`emails` には本文は保存されず、件名・送信元・宛先・受信日時などのメタデータのみが保存される。

## 目的
- `mailanalysis` が保存した `ParsedEmail` を入力に、`VendorResolution` により canonical な `Vendor` を解決できるようにする。
- `BillingEligibility` が生の `vendorName` ではなく、解決済み `Vendor` を前提に請求成立判定できるようにする。
- `Billing` の同一性である「ユーザー + Vendor + 請求番号」を安定化し、表記揺れや AI 解析の揺らぎによる重複・誤判定を防ぐ。
- `manualmailworkflow` に `billing` stage を追加できる前提を整理し、`fetch / analysis / billing` の責務境界を崩さないようにする。

## 機能要件
- `billing` は `manualmailworkflow` から渡される `user_id` と `parsed_email_ids` を入力として処理できること。
- `billing` は各 `parsed_email_id` から対象 `ParsedEmail` を読み込み、対応する元 `Email` を取得できること。
- `VendorResolution` は `ParsedEmail.vendorName` を候補値として扱い、canonical `Vendor` と同一視しないこと。
- `VendorResolution` は少なくとも以下の保存済み情報を使って解決できること。
  - `ParsedEmail` の候補値
  - 元 `Email` の件名
  - 元 `Email` の送信元
  - 元 `Email` の宛先
  - 元 `Email` の provider / account identifier / 受信日時などのメタデータ
- `VendorResolution` は保存済みデータだけで再実行可能であること。
  - `emails` に本文は保存されていないため、本文依存の解決を必須にしない。
- `VendorResolution` は外部 AI 再呼び出しではなく、決定的な解決処理として扱うこと。
- `VendorResolution` の結果は少なくとも「解決済み」「未解決」を区別できること。
- `Vendor` が未解決のときは `Billing` を生成しないこと。
- `BillingEligibility` は `VendorResolution` の結果を受けて成立判定を行うこと。
- 請求成立条件を満たした場合のみ `Billing` を生成すること。
- `Billing` 生成前に「ユーザー + Vendor + 請求番号」で重複確認を行うこと。
- 既存請求がある場合は重複作成せずスキップできること。
- 1 回の workflow 実行内で複数 `ParsedEmail` を処理でき、部分成功・部分失敗を扱えること。
- workflow は `billing` stage の結果として、作成件数・重複スキップ件数・vendor 未解決件数・請求不成立件数を集計できること。
- `billing.UseCase.Execute` は、vendor 未解決だったメールの `external_message_id` 一覧を重複なく返却できること。
- vendor 解決ルールは将来の alias 追加や既知マッピング拡張に耐えられること。

## 非機能要件
- 責務配置は既存方針どおり `billing` に置き、`mailanalysis` へ逆流させないこと。
- `mailanalysis` の責務は引き続き `ParsedEmail` 保存までに限定し、canonical Vendor 解決を持ち込まないこと。
- `VendorResolution` は同じ入力に対して同じ解決結果を返す、再現可能な処理であること。
- rerun 時に同じ `parsed_email_id` 群から同じ `Vendor` 解決結果と同じ重複判定が得られること。
- 保存済みメール本文が存在しない前提でも成立する設計にすること。
- ログやレスポンスにメール本文や秘匿情報を不要に露出しないこと。
- 監査や障害調査のため、少なくとも vendor 未解決・請求不成立・重複スキップの理由を識別できること。
- package 間依存は既存の Clean Architecture 方針に従い、application port 経由で接続すること。

## 制約事項
- 現行 `manualmailworkflow` は `fetch -> analysis` までしか実装されておらず、`billing` stage の追加が前提となる。
- 現行リポジトリには `billing` package と vendor 永続化実装がまだ存在しない。
- `internal/common/domain` には `Vendor`, `VendorResolution`, `BillingEligibility`, `Billing` の基本モデルはあるが、解決ルール本体は未実装である。
- `emails` テーブルには本文が保存されないため、`VendorResolution` は本文なしでも動作しなければならない。

## 確定事項
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
- `billing.UseCase.Execute` は vendor 未解決だったメールの `external_message_id` 一覧を重複なく返却する。
- `vendor_aliases` は同じ `alias_type + normalized_value` を複数 vendor に持てる設計とする。
- alias 競合時は resolver が `created_at` の新しい alias を 1 件選び、その時点の vendor で `Billing` を作成する。
- 将来的にユーザーが alias を修正できる前提で、初期は alias 競合を unresolved にはしない。
- ユーザー単位の上書きルールは初期スコープに含めず、後続エンハンスとする。

## 成功条件
- 同一 vendor の表記揺れや送信元揺れが、同一 canonical `Vendor` に正規化される。
- `Vendor` 未解決の `ParsedEmail` からは `Billing` が生成されない。
- `BillingEligibility` が解決済み `Vendor` を前提に評価される。
- 同一ユーザー内で同一 `Vendor + 請求番号` の `Billing` が重複作成されない。
- `manualmailworkflow` から `billing` stage を呼び出せる前提が整理される。
- 将来 alias や既知 vendor マッピングを増やしても、`mailanalysis` 側の責務を変更せず改善できる。

## 非スコープ
- vendor 手動補正 UI / API
- vendor alias 管理画面
- バッチ workflow への適用
- LLM による再解析や再推定
- vendor 解決結果そのものの永続化方式の最終確定
- ユーザー単位の vendor 上書きルール

## 未確定事項
