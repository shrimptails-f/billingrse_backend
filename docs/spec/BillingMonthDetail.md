# Billing Month Detail API 仕様

本ドキュメントは、請求サマリ画面で選択月の合計と支払先別内訳を表示するための API 仕様をまとめたものである。

参照元:
- `docs/api_design.md`
- `docs/architecture.md`
- `docs/ddd/README.md`
- `docs/ddd/invariants.md`

## 1. 概要

### 背景
- 既存実装では `GET /api/v1/billings` により請求一覧を取得できる。
- 一方で請求サマリ画面では、「選択月の合計」と「支払先別内訳」を軽く取得したい。
- DDD では `月次集計` を domain 概念として定義していないため、本 API は `Billing` を元にした read API として設計する。
- 12 ヶ月推移と選択月内訳を 1 endpoint にまとめると不要な vendor 集計まで毎回取得することになるため、本 API は単月詳細に責務を限定する。

### 目的
- 認証済みユーザーが、自分自身の請求データを指定月単位で詳細集計して取得できるようにする。
- 本 API は指定月の totals と vendor breakdown を返す。
- 月の所属判定は `billing_date` を優先し、未設定時のみ `received_at` を fallback とする。
- 初期表示では Monthly Trend API と並列に呼び出せる形にし、選択月変更時はこの API のみを再取得すればよい。

### 非スコープ
- 12 ヶ月ぶんの月別一覧
- 任意期間長の集計
- 通貨横断の 1 レスポンス返却
- vendor 詳細 API

## 2. API 契約

### Endpoint
- Method: `GET`
- Path: `/api/v1/billings/summary/monthly-detail/:year_month`
- Auth: required

### Path
- `year_month`
  - 必須
  - 形式は `YYYY-MM`

### Query
- `currency`
  - 任意
  - `JPY` | `USD`
  - default は `JPY`

### Response 200
```json
{
  "year_month": "2026-03",
  "currency": "JPY",
  "total_amount": 182400,
  "billing_count": 12,
  "fallback_billing_count": 3,
  "vendor_limit": 5,
  "vendor_items": [
    {
      "vendor_name": "AWS",
      "total_amount": 82000,
      "billing_count": 4,
      "is_other": false
    },
    {
      "vendor_name": "Google Workspace",
      "total_amount": 36000,
      "billing_count": 2,
      "is_other": false
    },
    {
      "vendor_name": "OpenAI",
      "total_amount": 24000,
      "billing_count": 2,
      "is_other": false
    },
    {
      "vendor_name": "Notion",
      "total_amount": 15000,
      "billing_count": 1,
      "is_other": false
    },
    {
      "vendor_name": "GitHub",
      "total_amount": 11200,
      "billing_count": 1,
      "is_other": false
    },
    {
      "vendor_name": "その他",
      "total_amount": 14200,
      "billing_count": 2,
      "is_other": true
    }
  ]
}
```

### Response field
- `year_month`
  - 対象月
- `currency`
  - 集計対象通貨
- `total_amount`
  - 対象月の請求総額
- `billing_count`
  - 対象月の請求件数
- `fallback_billing_count`
  - `billing_date` がなく、`received_at` で月判定した件数
- `vendor_limit`
  - 個別 vendor として返す件数上限
  - v1 は `5`
- `vendor_items`
  - 対象月の支払先別内訳
  - 合計金額降順
- `vendor_items[].vendor_name`
  - canonical vendor 名
  - rank 外の集約行は `"その他"`
- `vendor_items[].total_amount`
  - vendor 単位の合計金額
- `vendor_items[].billing_count`
  - vendor 単位の請求件数
- `vendor_items[].is_other`
  - `"その他"` 集約行かどうか

### Error
- `400 invalid_request`
  - `currency` が不正
  - `year_month` が不正
- `401 unauthorized`
  - JWT 不正または未認証
- `500 internal_server_error`
  - 集計取得の内部失敗

### 契約上の注意
- データが存在しない月も返す。
- `vendor_items` は空配列を返す。
- データが存在しない月は `total_amount=0`, `billing_count=0`, `fallback_billing_count=0`, `vendor_items=[]` を返す。
- 月判定ルールは固定であり、query で切り替えない。
- `billing_date` が存在する場合は `received_at` より優先する。
- v1 では `vendor_id` や vendor drill-down 用の追加情報は返さない。

## 3. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身が所有する `Billing` のみ集計対象になること。
- 通貨タブ切り替え用に 1 通貨単位で取得できること。
- `billing_date` 欠損件数を `fallback_billing_count` で確認できること。
- 各月の支払先別内訳は上位 5 件 + その他で返ること。
- 初期表示では Monthly Trend API と並列呼び出しできること。
- 選択月変更時はこの API だけを再取得すればよいこと。

## 4. 取得方針

### 月判定
- 集計対象日時は `billings.billing_summary_date` とする。
- `billing_summary_date` は保存時に `COALESCE(billings.billing_date, emails.received_at)` を materialize した派生カラムとする。
- `billing_date` を持つ請求はその月へ所属させる。
- `billing_date` が `null` の請求のみ `received_at` を使う。

### vendor 内訳
- vendor ごとに `SUM(amount)` と `COUNT(*)` を算出する。
- 合計金額降順で rank 付けし、6 位以下は `その他` へ合算する。

## 5. レイヤ設計

### Presentation
- `GET /api/v1/billings/summary/monthly-detail/:year_month`

### Application
- Month Detail 用 usecase を `internal/billingquery/application` に置く。
- top 5 + その他の整形は application 層で行う。

### Domain
- `Billing` aggregate は変更しない。
- 月次サマリは read model として扱う。

### Infrastructure
- `internal/billingquery/infrastructure` に `billings` と `vendors` を用いる summary repository を置く。
- 単月 totals と vendor breakdown を固定本数クエリで取得する。

## 6. 実装反映

- route:
  - `internal/app/router/router.go`
- DI:
  - `internal/di/billing.go`
- application:
  - `internal/billingquery/application/month_detail_usecase.go`
- infrastructure:
  - `internal/billingquery/infrastructure/month_detail_repository.go`
- presentation:
  - `internal/app/presentation/billing/month_detail_controller.go`

## 7. テスト観点

### Controller
- 200 正常系
- 400 invalid_request
- 401 unauthorized
- 500 internal_server_error

### UseCase
- 既定値適用
- `year_month` validation
- top 5 + その他
- fallback 件数の整形

### Repository
- `user_id` 所有範囲
- `currency` filter
- `billing_date` 優先 / `received_at` fallback
- 単月集計
- vendor 順序

### Router
- `GET /api/v1/billings/summary/monthly-detail/:year_month` が登録される

### 実装済みテストファイル
- `internal/app/presentation/billing/controller_test.go`
- `internal/billingquery/application/month_detail_usecase_test.go`
- `internal/billingquery/infrastructure/month_detail_repository_test.go`
- `internal/app/router/router_test.go`
