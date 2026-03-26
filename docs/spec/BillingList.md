# Billing 一覧 API 仕様

本ドキュメントは、`Billing` 一覧 API の要件定義・設計内容を 1 ファイルに集約した仕様書である。

参照元:
- `docs/api_design.md`
- `docs/architecture.md`
- `docs/ddd/README.md`
- `docs/ddd/invariants.md`

## 1. 概要

### 背景
- 現行実装では `billing` stage による請求生成・保存は存在する。
- 一方で、認証済みユーザーが自分の `Billing` を検索中心で一覧取得する HTTP API は未提供である。
- `Billing` の正本は `billings` テーブルにあるが、一覧表示には `Vendor` 名や参照元 `Email` の情報も必要になる。

### 目的
- 認証済みユーザーが、自分自身の請求一覧を検索中心で取得できるようにする。
- 検索対象日は `billing_date` を基準としつつ、請求日に欠損があるデータについては `Email.received_at` を fallback として扱えるようにする。
- 検索用の読み取り API として責務を限定し、請求更新・削除・エクスポートとは分離する。

### 非スコープ
- 請求の更新 API
- 請求の削除 API
- CSV / Excel などのエクスポート
- 並び順の自由指定
- `payment_cycle` や `currency` を使った追加フィルタ
- 請求詳細単体取得 API

## 2. API 契約

### Endpoint
- Method: `GET`
- Path: `/api/v1/billings`
- Auth: required

### Query
- `q`
  - 任意
  - 部分一致検索
  - 対象: `vendor_name`, `product_name_display`, `billing_number`, `external_message_id`
- `email_id`
  - 任意
  - 完全一致
- `external_message_id`
  - 任意
  - 完全一致
- `date_from`
  - 任意
  - RFC3339
- `date_to`
  - 任意
  - RFC3339
- `use_received_at_fallback`
  - 任意
  - `true|false`
  - default は `true`
- `limit`
  - 任意
  - default は `50`
  - max は `100`
- `offset`
  - 任意
  - default は `0`

### Response 200
```json
{
  "items": [
    {
      "email_id": 101,
      "external_message_id": "18f4c9b1...",
      "vendor_name": "AWS",
      "received_at": "2026-03-24T10:00:00Z",
      "billing_date": "2026-03-01T00:00:00Z",
      "product_name_display": "AWS Support Enterprise",
      "amount": 12345.678,
      "currency": "JPY"
    }
  ],
  "limit": 50,
  "offset": 0,
  "total_count": 132
}
```

### Response item
- `email_id`
  - 参照元 `Email` の ID
- `external_message_id`
  - 参照元 `Email` の外部メッセージ ID
- `vendor_name`
  - canonical `Vendor` の表示名
- `received_at`
  - 参照元 `Email` の受信日時
- `billing_date`
  - 請求日
  - `null` を許容する
- `product_name_display`
  - 表示用の商品名
  - `null` を許容する
- `amount`
  - 金額
  - JSON number とし、小数第 3 位までを許容する
- `currency`
  - ISO 4217 の 3 文字コード

### Error
- `400 invalid_request`
  - 不正な query parameter
  - `date_from > date_to`
  - `limit` / `offset` の範囲外
- `401 unauthorized`
  - JWT 不正または未認証
- `500 internal_server_error`
  - 一覧取得の内部失敗

### 契約上の注意
- JSON フィールド名は `lower_snake_case` とする。
- コレクションレスポンスは `items` を基本キーとする。
- `external_id` ではなく、既存用語に合わせて `external_message_id` を使う。
- v1 は検索中心 API のため、ページネーションは `limit` / `offset` に固定する。
- v1 は一覧 API に責務を限定し、無制限取得やエクスポート用途の契約にはしない。

## 3. 検索仕様

### 基本方針
- 全ての検索条件は AND で組み合わせる。
- `q` は trim 後に使い、空文字なら未指定として扱う。
- `q` は大文字小文字を区別しない部分一致検索とする。

### 日付検索の意味
本 API における検索対象日は、`billing_date` を基準とする。

- `use_received_at_fallback=true`
  - 検索対象日と並び順の基準は `billings.billing_summary_date` とする。
  - 請求日が欠損しているレコードも、受信日時ベースで検索できる。
- `use_received_at_fallback=false`
  - 日付検索には `billings.billing_date` のみを使う。
  - `billing_date IS NULL` のレコードは、`date_from` または `date_to` が指定された場合はヒットしない。

### 日付範囲の評価
- `date_from` は以上条件として扱う。
- `date_to` は以下条件として扱う。
- 比較に使う日時は RFC3339 の UTC として扱う。

### 並び順
既定の並び順は以下とする。

- `use_received_at_fallback=true`
  - `billing_summary_date DESC`
  - `billings.id DESC`
- `use_received_at_fallback=false`
  - `billing_date DESC NULLS LAST`
  - `emails.received_at DESC`
  - `billings.id DESC`

補足:
- `billing_summary_date` は API の公開フィールドではなく、検索・並び順のための内部概念とする。
- `billing_summary_date` は保存時に `COALESCE(billings.billing_date, emails.received_at)` を materialize した派生カラムとする。
- offset pagination でも順序がぶれにくいよう、最後に `billings.id` を tie-breaker に使う。

### ページネーション
- `total_count` は filter 適用後、`limit` / `offset` 適用前の件数とする。
- `limit` 未指定時は `50` を適用する。
- `offset` 未指定時は `0` を適用する。
- 大きな offset の性能最適化は v1 のスコープ外とする。

## 4. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身が所有する `Billing` のみ取得できること。
- レスポンス各 item は少なくとも以下を返せること。
  - `email_id`
  - `external_message_id`
  - `vendor_name`
  - `received_at`
  - `billing_date`
  - `product_name_display`
  - `amount`
  - `currency`
- `billing_date` が欠損している請求も一覧対象に含められること。
- fallback の有無を query parameter で切り替えられること。
- ページ単位で件数制御できること。
- レスポンスや構造化ログにメール本文など不要な秘匿情報を含めないこと。

## 5. 取得方針

本 API は検索用の read API とし、`Billing` aggregate をそのまま返さず、一覧表示向け read model を返す。

### 取得アルゴリズム
1. 認証コンテキストから `user_id` を取得する。
2. `billings` を起点に `vendors` と `emails` を join する。
3. `billings.user_id = current_user_id` で絞る。
4. `email_id`、`external_message_id`、`q`、日付条件を適用する。
5. 規定の並び順で `limit` / `offset` を適用する。
6. `items` と `total_count` を返す。

### 補足
- `vendor_name` は canonical `Vendor` の `name` を使う。
- `received_at` と `external_message_id` は参照元 `Email` から返す。
- `billing_summary_date` は `use_received_at_fallback=true` の検索・並び順専用の内部カラムであり、レスポンスには含めない。
- `Billing` の参照元は `Email` であり、`ParsedEmail` はレスポンスに含めない。
- v1 の `q` は通常の部分一致検索とし、全文検索や検索専用 index 最適化は別タスクとする。

## 6. レイヤ設計

### Presentation
- `GET /api/v1/billings` を追加する。
- handler は `(*Controller).List` とする。
- query DTO は controller ファイル内に置く。
- controller は認証済み `userID` を受け取り、application 層へ検索条件を渡して response DTO に変換する。

### Application
- `internal/billingquery/application` に一覧取得 usecase を置く。
- `ListQuery`, `ListResult`, `ListItem`, `BillingListRepository` を定義する。
- query parameter の normalize / validate をここで扱う。
- read model は一覧 API 用に閉じた shape とし、`common/domain.Billing` aggregate は直接返さない。

### Domain
- 既存の `Billing` aggregate と不変条件は維持する。
- 一覧 API は参照系のため、`Vendor` 名や `Email` 情報を含む read model を別途持つ。
- 一覧 API のために `Billing` aggregate 自体へ表示項目を過剰に持ち込まない。

### Infrastructure
- `internal/billingquery/infrastructure` に、`billings`, `vendors`, `emails` を join する read repository を置く。
- 並び順と filter のロジックは SQL で安定して表現する。
- `total_count` と `items` を同一条件で取得できるようにする。

## 7. 実装反映

- route:
  - `internal/app/router/router.go`
- DI:
  - `internal/di/billing.go`
- application:
  - `internal/billingquery/application/list_usecase.go`
- infrastructure:
  - `internal/billingquery/infrastructure/list_repository.go`
- presentation:
  - `internal/app/presentation/billing/controller.go`

### 実装済みテストファイル
- `internal/app/presentation/billing/controller_test.go`
- `internal/billingquery/application/list_usecase_test.go`
- `internal/billingquery/infrastructure/list_repository_test.go`
- `internal/app/router/router_test.go`

## 8. テスト観点

### Controller
- 200 正常系
- 400 invalid_request
- 401 unauthorized
- 500 internal_server_error

### UseCase
- query normalize / validate
- fallback `true` / `false` の切り替え
- `date_from > date_to` の拒否
- `limit` / `offset` の既定値適用

### Repository
- `user_id` で所有範囲が絞られること
- `q` の部分一致検索
- `email_id` / `external_message_id` の完全一致
- fallback `true` / `false` で日付検索結果が変わること
- 並び順が安定していること
- `total_count` が filter 適用後件数になっていること

### Router
- `GET /api/v1/billings` が登録される

## 9. 判断事項

- v1 は検索中心 API とし、エクスポート用途にはしない。
- v1 は `limit` / `offset` を採用する。
- v1 は並び順の自由指定を入れず、固定の規定順のみとする。
- 日付検索の fallback は default `true` とする。
- v1 のレスポンスはユーザー要件に合わせ、`billing_id` を含めない。
