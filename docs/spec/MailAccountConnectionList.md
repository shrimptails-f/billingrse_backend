# MailAccountConnection 一覧 API 仕様

本ドキュメントは、`MailAccountConnection` 一覧 API の最新の要件定義・設計内容を 1 ファイルに集約した仕様書である。

参照元:
- `tasks/20260320_mail_account_connection一覧状態api/要件定義.md`
- `tasks/20260320_mail_account_connection一覧状態api/設計.md`

## 1. 概要

### 背景
- 既存実装では Gmail OAuth の開始 API と callback API は存在する。
- 一方で、ユーザーが自分の連携済みメールアカウントを一覧する API は未提供。
- DDD 上の公開概念は `MailAccountConnection` だが、現行実装では `email_credentials` テーブルを backing store として利用している。

### 目的
- 認証済みユーザーが、自分自身のメール連携一覧を取得できるようにする。
- 本 API は「設定済みの連携一覧」を返すことに責務を限定し、外部 provider への有効確認は行わない。
- 現状は Gmail のみ対象だが、将来的な Outlook 等の追加に耐える HTTP 契約と責務分割にする。

### 非スコープ
- 連携状態のリアルタイム有効確認
- 連携解除 API
- 連携詳細単体取得 API
- pending OAuth state の一覧表示
- バッチ設定やメール取得実行 API
- Outlook 実装そのもの

## 2. API 契約

### Endpoint
- Method: `GET`
- Path: `/api/v1/mail-account-connections`
- Auth: required
- Query: なし

### Response 200
```json
{
  "items": [
    {
      "id": 12,
      "provider": "gmail",
      "account_identifier": "user@gmail.com",
      "created_at": "2026-03-19T12:34:56Z",
      "updated_at": "2026-03-19T12:40:12Z"
    }
  ]
}
```

### Response item
- `id`
  - connection record ID
- `provider`
  - `gmail`, 将来 `outlook` など
- `account_identifier`
  - provider 非依存の識別子
  - Gmail では Gmail address
- `created_at`
  - 初回連携作成日時
- `updated_at`
  - 最終再連携日時

### Error
- `401 unauthorized`
  - JWT 不正または未認証
- `500 internal_server_error`
  - DB 読み出し失敗など、一覧のベースデータ取得に失敗した場合

### 契約上の注意
- JSON フィールド名は `lower_snake_case` とする。
- コレクションレスポンスは `items` を基本キーとする。
- token 値や digest はレスポンスに含めない。
- v1 ではページングを入れない。必要になった場合も `items` を維持したまま `next_cursor` などを追加する。

## 3. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身の `MailAccountConnection` のみ取得できること。
- レスポンスはレコード単位の一覧で返すこと。
- 各レコードに `provider` を含めること。
- 各レコードに provider 非依存の `account_identifier` を含めること。
- 各レコードに `created_at` と `updated_at` を含めること。
- レスポンスや構造化ログにアクセストークン / リフレッシュトークン / digest / auth code を出さないこと。

## 4. 取得方針

本 API は connection の存在一覧を返すだけで、現在利用可能かどうかの判定は行わない。

### 取得アルゴリズム
1. `user_id` で `email_credentials` を一覧取得する。
2. 各レコードを provider 非依存の view model に変換する。
3. `items` 配列として返す。

### 補足
- pending OAuth state は `MailAccountConnection` ではないため本 API に含めない。
- access token / refresh token / digest は storage にのみ使い、レスポンスには含めない。
- `token_expiry` などの保持情報から状態推定も行わない。

## 5. レイヤ設計

### Presentation
- `GET /api/v1/mail-account-connections` を追加する。
- handler は `(*Controller).List` とする。
- HTTP DTO は controller ファイル内に置く。
- controller は認証済み `userID` を受け取り、application 層の一覧取得を呼び出して `items` を返す。

### Application
- `UseCaseInterface` に `ListConnections(ctx context.Context, userID uint) ([]ConnectionView, error)` を追加する。
- authorize / callback と同じ `emailcredential` usecase に置く。
- 一覧 API では provider への問い合わせやトークン復号は行わない。
- `email_credentials` から取得した storage model を provider 非依存の read model に変換する。

### Domain
- `EmailCredential` は storage model として維持する。
- 一覧 API 用に `ConnectionView` を追加する。
- `ConnectionView` の field は `ID`, `Provider`, `AccountIdentifier`, `CreatedAt`, `UpdatedAt` とする。
- 既存の `internal/common/domain.MailAccountConnection` は今回の API では再利用しない。

### Infrastructure
- Repository に `ListCredentialsByUser(ctx context.Context, userID uint) ([]domain.EmailCredential, error)` を追加する。
- DB の `gmail_address` 列は内部実装として当面許容し、application で `account_identifier` に写像する。
- 並び順は固定し、同一ユーザーの一覧を安定して返す。

## 6. Provider 拡張方針

- HTTP path は provider 非依存の collection path を維持する。
- response item は `provider` と `account_identifier` を持つため、Outlook 追加時も shape を変えない。
- 既存 DB の `gmail_address` 列は内部実装として当面許容する。
- 2 つ目の provider 実装着手前に `account_identifier` 系の汎用列へ寄せる migration を別タスクで行うのが望ましい。
- 将来、接続状態表示が必要になった場合は別 API または追加フィールドで扱う。v1 一覧 API の責務には含めない。

## 7. テスト観点

### Controller
- 200 正常系
- 401 未認証
- 500 ベース一覧取得失敗

### UseCase
- credential 0 件
- 複数件を provider 非依存 shape に変換できること
- 一覧取得失敗時に error を返すこと

### Repository
- user 単位の一覧取得
- 並び順の固定

### Router
- `GET /api/v1/mail-account-connections` が登録される

## 8. 判断事項

- v1 は単一一覧 API に集約し、単体 detail API は作らない。
- v1 は live check を行わない。
- 一覧 API は設定済み連携の存在を返すことに責務を限定する。
