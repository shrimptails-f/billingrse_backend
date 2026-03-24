# 手動メール取得アーキテクチャ 要件定義

## 背景
- DDD 上、`ManualMailFetch` は「手動トリガーで即時実行される Email 取得」であり、Email の意味解釈は持たない。
- 一方で実装上は、外部メールサービスからの取得、AI による字句解析、`ParsedEmail` 保存、`VendorResolution`、`BillingEligibility`、`Billing` 生成まで一連の流れが必要になる。
- この一連の処理を 1 package / 1 usecase に押し込むと、責務が肥大化し、テスト・拡張・障害切り分けが難しくなる。

## 目的
- `internal/mailfetch`
- `internal/mailanalysis`
- `internal/billing`

上記 3 package に責務を分割し、それぞれが何を行うかを明確にする。

## スコープ
- 手動メール取得 API 起点のバックエンド処理分割
- 外部メールサービス取得の境界定義
- AI 解析の境界定義
- `ParsedEmail` / `Billing` の保存責務整理
- `VendorResolution` と `BillingEligibility` の配置整理
- package 間の依存方向整理

## 非スコープ
- バッチメール取得
- Frontend 実装
- 永続化テーブルの最終確定
- ジョブキュー製品の選定
- Outlook など別メールサービスの具体実装

## 全体フロー
1. Presentation が手動メール取得リクエストを受ける。
2. `mailfetch` が連携情報と取得条件を検証し、外部メールサービスからメールを取得する。
3. `mailfetch` が取得した生メールを `Email` として保存する。
4. `manualmailworkflow` が `mailfetch` の返却した `created_emails` を `mailanalysis` へ渡す。
5. `mailanalysis` が `Email` を AI 解析し、`ParsedEmail` を生成・保存する。
6. `manualmailworkflow` が `mailanalysis` の返却した `parsed_email_ids` を `billing` へ渡す。
7. `billing` が `VendorResolution` と `BillingEligibility` を適用し、成立したものだけ `Billing` を生成・保存する。

## package ごとの責務

### `internal/mailfetch`
- 役割
  - 手動実行の開始点
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
  - 保存済み `parsed_email_ids` の返却
- やらないこと
  - 外部メールサービスからの取得
  - canonical Vendor の解決
  - `Billing` 生成

### `internal/billing`
- 役割
  - `ParsedEmail` と元 `Email` の読み込み
  - `VendorResolution` の適用
  - `BillingEligibility` の適用
  - 重複判定
  - `Billing` の生成・保存
- やらないこと
  - 外部メールサービスからの取得
  - AI 呼び出し

## 保存対象の整理
- `Email`
  - `mailfetch` が保存する
- `ParsedEmail`
  - `mailanalysis` が保存する
- `Billing`
  - `billing` が保存する
- `VendorResolution`
  - 原則としてポリシーであり、永続化対象の本体ではない
  - 監査が必要なら「解決結果のスナップショット」を別途保存する
- `BillingEligibility`
  - 永続化しない判断モデル
  - 監査が必要なら「判定結果と理由」を別途保存する

## 設計方針
- package 間は直接具体実装に依存せず、application 層の port 経由で連携する。
- 外部メールサービスと AI は adapter で隠蔽する。
- provider 切替点では factory を使う。
- Go では abstract class は使わず、小さい interface で責務を分ける。
- HTTP は `manualmailworkflow` を直接呼び、以降の段は workflow 内で接続する。

## 成功条件
- `mailfetch / mailanalysis / billing` の責務境界がチーム内で共有できる。
- `ManualMailFetch` が意味解釈を持たないことが設計上も維持される。
- `ParsedEmail` と `Billing` の保存責務が別 package に整理される。
- `VendorResolution` と `BillingEligibility` が billing 側の判断責務として整理される。
- 今後 Gmail 以外の provider 追加や AI provider 切替が adapter/factory 追加で対応可能になる。
