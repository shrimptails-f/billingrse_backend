# Billing Monthly Trend API 仕様

本ドキュメントは、請求サマリ画面で直近 12 ヶ月の推移を表示するための API 仕様をまとめたものである。

参照元:
- `docs/api_design.md`
- `docs/architecture.md`
- `docs/ddd/README.md`
- `docs/ddd/invariants.md`

## 1. 概要

### 背景
- 既存実装では `GET /api/v1/billings` により請求一覧を取得できる。
- 一方で請求サマリ画面では、一覧ではなく「直近 12 ヶ月の推移」を軽く取得したい。
- DDD では `月次集計` を domain 概念として定義していないため、本 API は `Billing` を元にした read API として設計する。
- 選択月の vendor 内訳は別 API に分離し、本 API は 12 ヶ月の totals 集計に責務を限定する。

### 目的
- 認証済みユーザーが、自分自身の請求データを通貨別に月次集計して取得できるようにする。
- 本 API は月別 totals のみを返し、vendor breakdown は返さない。
- 月の所属判定は `billing_date` を優先し、未設定時のみ `received_at` を fallback とする。
- 返却する `default_selected_month` は、同時に呼ぶ Month Detail API の初期対象月として使える値とする。

### 非スコープ
- vendor 内訳
- 任意期間長の集計
- 通貨横断の 1 レスポンス返却
- 日 / 週単位の集計

## 2. API 契約

### Endpoint
- Method: `GET`
- Path: `/api/v1/billings/summary/monthly-trend`
- Auth: required

### Query
- `currency`
  - 任意
  - `JPY` | `USD`
  - default は `JPY`
- `window_end_month`
  - 任意
  - 形式は `YYYY-MM`
  - default はリクエスト時点の UTC 現在月

### Response 200
```json
{
  "currency": "JPY",
  "window_start_month": "2025-04",
  "window_end_month": "2026-03",
  "default_selected_month": "2026-03",
  "items": [
    {
      "year_month": "2025-04",
      "total_amount": 0,
      "billing_count": 0,
      "fallback_billing_count": 0
    },
    {
      "year_month": "2026-03",
      "total_amount": 182400,
      "billing_count": 12,
      "fallback_billing_count": 3
    }
  ]
}
```

### Response field
- `currency`
  - 集計対象通貨
- `window_start_month`
  - 12 ヶ月 window の開始月
- `window_end_month`
  - 12 ヶ月 window の終了月
- `default_selected_month`
  - 初期選択に使う月
  - v1 は `window_end_month` と同値
- `items`
  - Monthly Trend 配列
  - 常に 12 件
- `items[].year_month`
  - `YYYY-MM`
- `items[].total_amount`
  - その月の請求総額
- `items[].billing_count`
  - その月の請求件数
- `items[].fallback_billing_count`
  - `billing_date` がなく、`received_at` で月判定した件数

### Error
- `400 invalid_request`
  - `currency` が不正
  - `window_end_month` が不正
- `401 unauthorized`
  - JWT 不正または未認証
- `500 internal_server_error`
  - 集計取得の内部失敗

### 契約上の注意
- `items` は開始月から終了月まで昇順で返す。
- データが存在しない月も返す。
- データが存在しない月は `total_amount=0`, `billing_count=0`, `fallback_billing_count=0` の bucket を返す。
- 月判定ルールは固定であり、query で切り替えない。
- `billing_date` が存在する場合は `received_at` より優先する。

## 3. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身が所有する `Billing` のみ集計対象になること。
- 通貨タブ切り替え用に 1 通貨単位で取得できること。
- `billing_date` 欠損件数を `fallback_billing_count` で確認できること。
- 12 ヶ月 zero-fill されたレスポンスを返せること。
- 初期表示では Month Detail API と並列呼び出しできること。

## 4. 取得方針

### 月判定
- 集計対象日時は `billings.billing_summary_date` とする。
- `billing_summary_date` は保存時に `COALESCE(billings.billing_date, emails.received_at)` を materialize した派生カラムとする。
- `billing_date` を持つ請求はその月へ所属させる。
- `billing_date` が `null` の請求のみ `received_at` を使う。

### 集計 window
- `window_end_month` 基準の直近 12 ヶ月を返す。

## 5. レイヤ設計

### Presentation
- `GET /api/v1/billings/summary/monthly-trend`
- controller 実装は `internal/app/presentation/billing/monthly_trend_controller.go`
- handler は `(*Controller).MonthlyTrend`

### Application
- Monthly Trend 用 usecase を `internal/billingquery/application` に置く。
- zero-fill は application 層で行う。
- 実装上の主要型:
  - `MonthlyTrendQuery`
  - `MonthlyTrendAggregate`
  - `MonthlyTrendResult`
  - `MonthlyTrendUseCase`
- `currency` と `window_end_month` の既定値適用は usecase で行い、現在月判定は injected clock を使う。

### Domain
- `Billing` aggregate は変更しない。
- 月次サマリは read model として扱う。

### Infrastructure
- `internal/billingquery/infrastructure` に `billings` を起点とする summary repository を置く。
- 月別 totals を 1 クエリで取得する。
- repository 実装は `internal/billingquery/infrastructure/monthly_trend_repository.go`
- 集計は `billings.billing_summary_date` を基準に `YEAR()` / `MONTH()` で group 化し、`YYYY-MM` 文字列へ整形して返す。

## 6. 実装反映

- route:
  - `internal/app/router/router.go`
- DI:
  - `internal/di/billing.go`
- application:
  - `internal/billingquery/application/monthly_trend_usecase.go`
- infrastructure:
  - `internal/billingquery/infrastructure/monthly_trend_repository.go`
- presentation:
  - `internal/app/presentation/billing/monthly_trend_controller.go`

## 7. テスト観点

### Controller
- 200 正常系
- 400 invalid_request
- 401 unauthorized
- 500 internal_server_error

### UseCase
- 既定値適用
- 12 ヶ月 zero-fill
- fallback 件数の整形

### Repository
- `user_id` 所有範囲
- `currency` filter
- `billing_date` 優先 / `received_at` fallback
- 月別集計

### Router
- `GET /api/v1/billings/summary/monthly-trend` が登録される

### 実装済みテストファイル
- `internal/app/presentation/billing/controller_test.go`
- `internal/billingquery/application/monthly_trend_usecase_test.go`
- `internal/billingquery/infrastructure/monthly_trend_repository_test.go`
- `internal/app/router/router_test.go`
