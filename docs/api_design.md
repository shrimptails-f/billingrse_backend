# API 共通仕様 設計

## 目的

本ドキュメントは、このリポジトリで提供する HTTP API の設計方針を統一するためのルールを定義する。

- 実装・レビュー・保守の判断基準を明確にする
- 本設計書ではコーディング規約とアーキテクチャはスコープ外とする
- `internal/app/presentation` と `internal/app/middleware` における HTTP 境界の責務を揃える

## 適用範囲

- 対象:
  - `internal/app/presentation/**`
  - `internal/app/middleware/**`
  - `internal/app/router/**`
- 非対象:
  - Gmail / OpenAI などの外部 API クライアント仕様
  - 非同期メッセージング内部のインターフェース

## 基本方針

- HTTP API は外部契約であり、内部の domain / application / infrastructure の都合をそのまま露出しない
- Controller は「入力の受け取り」「アプリケーション層 DTO への変換」「HTTP ステータスとレスポンスへのマッピング」に責務を限定する

## 1. エンドポイント設計

### 1.1 パス設計

- ルートは責務単位でグルーピングする
  - 例: `/api/v1/auth`, `/api/v1/emails`, `/api/v1/billings`
- パスセグメントは lower-case ASCII を使う
- 複数語のセグメントが必要な場合のみ kebab-case を使う
- 末尾スラッシュは付けない

### 1.2 リソース指向を優先する

- 基本は名詞で設計する
  - コレクション: `/api/v1/emails`
  - 単体: `/api/v1/emails/:email_id`
- 動詞ベースのアクションは、リソースに自然に落とし込めない場合に限定する
  - 許容例: `/api/v1/auth/login`, `/api/v1/auth/logout`, `/api/v1/auth/register`
- サブリソースで意図が明確になる場合は、ネストで表現する
  - 例: `/api/v1/auth/email/verify`, `/api/v1/auth/email/resend`

### 1.3 HTTP メソッドの意味

- `GET`
  - 取得系
- `POST`
  - データ作成、無効化、解除
  - サーバー主導の副作用を持つアクション
- `PUT`
  - 全置換
- `PATCH`
  - 部分更新
- `DELETE`
  - 削除

注意:
認証メールの正規入口は frontend の `/signup/verify` とする。backend では `POST /api/v1/auth/email/verify` のみが状態変更の本体であり、メール本文には frontend の URL を記載する。

### 1.4 Path / Query / Body の使い分け

- Path parameter:
  - リソース識別子
- Query parameter:
  - 検索条件
  - ソート
  - ページング
- JSON body:
  - `POST` / `PUT` / `PATCH` の入力
  - `POST /api/v1/auth/email/verify` のような状態変更アクションの入力
- 機微情報は原則 body または header で受け取り、Query には載せない
  - frontend の画面遷移用に query で token を扱う場合でも、backend API への送信は body へ詰め替える

### 1.5 バージョニング

- 現時点の API は `/api/v1/...` を正規契約とする
- 破壊的変更が避けられない段階で、パス先頭のバージョンを更新する
  - 例: `/api/v2/...`
- 一時的な unversioned compatibility endpoint を残す場合でも、現行契約としては扱わない
- payload 内に `version` フィールドを混在させて分岐しない

## 2. リクエスト設計

### 2.1 DTO 定義

- HTTP の request / response DTO は feature 配下の controller ファイルに置く
- JSON の shape が domain DTO と完全一致しない限り、HTTP 専用 DTO を定義する
- JSON フィールド名は `lower_snake_case` に統一する

### 2.2 バリデーション

- JSON リクエストは `ShouldBindJSON` を使う
- 必須項目、メール形式などの構文バリデーションは `binding` タグで明示する
- Controller は構文バリデーションを担当し、業務ルールの検証は application 層へ委譲する

## 3. 成功レスポンス設計

### 3.1 基本ルール

- 成功時は endpoint ごとの目的に沿った JSON を返す
- 汎用的な レスポンスフォーマットは導入しない
- リソース返却か、アクション結果返却かが分かる最小構造にする

### 3.2 ステータスコード

| ステータス | 用途 |
| --- | --- |
| `200 OK` | 取得成功、または結果本文を返すアクション成功 |
| `202 Accepted` | 非同期処理の受付成功。処理完了は別 API や後続取得で確認する |
| `201 Created` | 新規作成成功 |
| `204 No Content` | 本文不要な成功。Cookie / Header のみ返す操作を含む |

### 3.3 レスポンス形

- 単一リソース:
  - 例: `{"id":1,"email":"user@example.com"}`
- アクション結果:
  - 例: `{"message":"確認メールを再送信しました。"}`
- 複合結果:
  - 例: `{"message":"登録が完了しました。","user":{...}}`
- 非同期受付結果:
  - 例: `{"message":"処理を受け付けました。","workflow_id":"01HQ...","status":"queued"}`
- コレクション:
  - `items` を基本キーとし、必要に応じて `next_cursor`, `total_count` を追加する
- `204 No Content` のときは空ボディにする
- 日時は Go 標準の RFC3339 形式を前提とする
- 値が存在しない日時や任意項目は `null` を許容する

非同期アクションの補足:
- `202 Accepted` を返す endpoint は「受付完了」を示すだけで、処理成功を意味しない。
- 原則としてクライアントが後続取得すべき識別子（例: `workflow_id`, `job_id`）を返す。
- 完了結果は原則として別の `GET` endpoint で取得させる。
- 状態取得 API をまだ公開しない場合は、返した識別子の用途を相関 ID に限定し、その例外を feature spec / architecture 側で明示する。

## 4. エラーレスポンス設計

### 4.1 標準形式

エラーは次の形に統一する。

```json
{
  "error": {
    "code": "invalid_request",
    "message": "入力値が不正です。"
  }
}
```

バリデーションの詳細が必要な場合のみ `details` を追加してよい。

注記:
- 現行の `internal/app/httpresponse` 共通実装は `code` / `message` までを標準化しており、`details` はまだ共通 helper 化されていない。
- `details` を契約として返す API を追加する場合は、endpoint 個別実装で済ませず共通レスポンス実装も合わせて拡張する。

```json
{
  "error": {
    "code": "invalid_request",
    "message": "入力値が不正です。",
    "details": [
      {
        "field": "email",
        "reason": "required"
      }
    ]
  }
}
```

### 4.2 各フィールドのルール

- `error.code`
  - クライアント向けの安定した機械可読コード
  - `snake_case` を使う
  - Go の error 文字列をそのまま返さない
- `error.message`
  - 画面表示しても問題ない利用者向け文言
  - 既定値は日本語
  - 内部実装、SQL、外部 API の生エラーを含めない
- `error.details`
  - 任意
  - 入力項目単位の補足が必要な場合のみ返す

### 4.3 返却方針

- `4xx` は、クライアントが修正可能な失敗理由を返す
- `5xx` は、原因を秘匿した一般化メッセージを返す
- `4xx` で空ボディを返さない
- middleware も controller と同じエラー形式に揃える

### 4.4 ステータスと意味

| ステータス | 用途 |
| --- | --- |
| `400 Bad Request` | JSON 不正、必須不足、クエリ不足、業務前提を満たさない入力 |
| `401 Unauthorized` | 認証情報なし、または無効 |
| `403 Forbidden` | 認証済みだが実行権限または状態条件を満たさない |
| `404 Not Found` | 対象リソースなし |
| `409 Conflict` | 一意制約や同時更新衝突など、競合として区別する価値がある場合 |
| `429 Too Many Requests` | レート制限超過 |
| `500 Internal Server Error` | 想定外エラー |
| `503 Service Unavailable` | 一時的な外部依存障害で、再試行判断を促したい場合 |

認証系の代表例:
- `email_already_exists` は `409 Conflict`
- `already_verified` は `403 Forbidden`
- `token_expired` と `token_already_used` は `409 Conflict`
- `mail_send_failed` は `503 Service Unavailable`

## 5. テストルール

- Controller / Middleware テストでは最低限次を検証する
  - 正常系のステータスと本文
  - 構文バリデーション失敗時のステータスとエラー形式
  - 代表的な業務エラーのマッピング
  - Cookie / Header の契約
  - 認証失敗時のステータスとエラー形式
- `204 No Content` を返す API は、本文が空であることもテストする
- API 契約を変える変更では、成功ケースだけでなく失敗ケースのテストも更新する
