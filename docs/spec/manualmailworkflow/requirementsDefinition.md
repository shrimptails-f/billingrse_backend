# 手動メール取得アーキテクチャ 要件定義

## 背景
- DDD 上、`ManualMailFetch` は「手動トリガーで即時実行される Email 取得」であり、Email の意味解釈は持たない。
- 一方で実装上は、外部メールサービスからの取得、AI による字句解析、`ParsedEmail` 保存、`VendorResolution`、`BillingEligibility`、`Billing` 生成まで一連の流れが必要になる。
- 現行の同期実行 API では、HTTP リクエストが `mailfetch` や `mailanalysis` の完了まで待つため、応答が長時間化しやすい。
- 今回は `manualmailworkflow` を非同期受付型へ切り替え、`POST` はすぐ返し、実処理はバックグラウンドで進める前提に改める。

## 目的
- 手動メール取得 API を非同期実行へ変更し、HTTP では受付結果だけを短時間で返せるようにする。
- `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` の実処理は background workflow として順次実行できるようにする。
- フロントエンドが workflow の進行状態と最終結果を別 API で取得できるようにする。

## スコープ
- 手動メール取得 API の非同期化
- workflow 受付 API と、将来追加する状態取得 API の責務定義
- background workflow における stage 実行順序の定義
- package 間の依存方向整理
- フロントエンド連携に必要な API 契約整理

## 非スコープ
- バッチメール取得
- ジョブキュー製品の最終選定
- workflow 履歴テーブル / 結果保存テーブルの詳細設計
- Frontend 実装そのもの
- Outlook など別メールサービスの具体実装

## 全体フロー
1. Presentation が手動メール取得リクエストを受ける。
2. `manualmailworkflow` が入力を検証し、workflow 識別子を採番して受付状態を作る。
3. `manualmailworkflow` が background 実行を dispatcher に依頼する。
4. HTTP は `202 Accepted` と `workflow_id` を返して終了する。
5. background runner が `mailfetch` を実行し、`Email` を取得・保存する。
6. background runner が `mailanalysis` を実行し、`ParsedEmail` を生成・保存する。
7. background runner が `vendorresolution` を実行し、canonical `Vendor` を解決する。
8. background runner が `billingeligibility` を実行し、請求成立可否を判定する。
9. background runner が `billing` を実行し、`Billing` を生成・保存する。
10. クライアントは `workflow_id` を相関 ID として保持し、状態取得 API を追加する場合はそれを使って進行状態または最終結果を参照する。

補足:
- workflow 履歴の永続化詳細は今回深掘りせず、`TODO:` として残す。
- stage 間のデータ受け渡しは、既存方針どおり workflow payload を優先する。

## package ごとの責務

### `internal/manualmailworkflow`
- 役割
  - workflow 開始 request の受付
  - `workflow_id` 採番と受付結果返却
  - background 実行の dispatch
  - workflow 状態取得
  - stage 実行結果の集約
- やらないこと
  - 各 stage の個別業務ロジック実装
  - Gmail / OpenAI / Gorm の直接呼び出し
  - workflow 履歴テーブル詳細の最終決定

### `internal/mailfetch`
- 役割
  - `MailAccountConnection` の確認
  - `FetchCondition` の検証
  - 外部メールサービスからの取得
  - `Email` の idempotent 保存
  - 保存済み `created_email_ids` / `created_emails` / `existing_email_ids` の返却
- やらないこと
  - AI 解析
  - `VendorResolution`
  - `BillingEligibility`
  - `Billing` 生成

### `internal/mailanalysis`
- 役割
  - `mailfetch` から渡された `created_emails` の利用
  - AI 呼び出し
  - AI 応答の構造化
  - `ParsedEmail` の解析履歴保存
  - 保存済み `parsed_email_ids` と workflow 向け `parsed_emails` payload の返却
- やらないこと
  - 外部メールサービスからの取得
  - canonical Vendor の解決
  - `BillingEligibility` 判定
  - `Billing` 生成

### `internal/vendorresolution`
- 役割
  - workflow payload として渡された `ParsedEmail` / source email 情報の利用
  - vendor master / alias など判定に必要な facts の取得
  - `VendorResolution` の適用
  - canonical `Vendor` の解決
  - resolved / unresolved の返却
- やらないこと
  - AI 呼び出し
  - `BillingEligibility` 判定
  - `Billing` 生成
  - `Billing` 保存

### `internal/billingeligibility`
- 役割
  - `VendorResolution` の結果を受け取る
  - workflow payload で受け取った `ParsedEmail` 情報の利用
  - 請求成立条件を評価する
  - eligible / ineligible の返却
- やらないこと
  - canonical Vendor の解決
  - `Billing` 保存

### `internal/billing`
- 役割
  - `BillingEligibility` の結果を受け取る
  - `Billing` の生成
  - 重複判定を含む idempotent 保存
  - created / duplicate / failure の結果返却
- やらないこと
  - 外部メールサービスからの取得
  - AI 呼び出し
  - canonical Vendor の解決
  - 請求成立判定

## 保存対象の整理
- `Email`
  - `mailfetch` が保存する
- `ParsedEmail`
  - `mailanalysis` が保存する
- `Billing`
  - `billing` が保存する
- workflow 受付状態 / 最終結果
  - `manualmailworkflow` が扱う
  - `TODO:` 履歴テーブルおよび保存形式の詳細設計

## 設計方針
- `POST /api/v1/manual-mail-workflows` は workflow 完了を待たず、受付結果のみを返す。
- 完了結果の参照 API は将来追加対象とし、現時点では `workflow_id` を background 実行の相関 ID として返す。
- HTTP request 中に保持するのは入力検証・受付・dispatch までとし、実処理は background context で進める。
- package 間は直接具体実装に依存せず、application 層の port 経由で接続する。
- stage 間連携は workflow payload を優先し、不要な再読込は増やさない。
- `billing` の duplicate 判定は `Exists + Save` の 2 段階ではなく、DB 一意制約を前提にした idempotent 保存として設計する。

## 成功条件
- 手動メール取得 API が短時間で `202 Accepted` を返せる。
- workflow が background で `mailfetch -> mailanalysis -> vendorresolution -> billingeligibility -> billing` の順に実行される。
- フロントエンドが `workflow_id` を相関 ID として扱え、状態取得 API を追加する場合は進行状態または最終結果を取得できる。
- 各 stage の責務境界が崩れず、既存 stage 実装を流用できる。
- workflow 履歴保存の詳細が未確定でも、非同期 API 契約の整理が先に進められる。
