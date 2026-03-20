# MailAccountConnection 連携解除 API 仕様

本ドキュメントは、`MailAccountConnection` 連携解除 API の最新の要件定義・設計内容を 1 ファイルに集約した仕様書である。

参照元:
- `tasks/20260320_mail_account_connection連携解除api/要件定義.md`
- `tasks/20260320_mail_account_connection連携解除api/設計.md`

## 1. 概要

### 背景
- 既存実装では Gmail OAuth の開始 API、callback API、連携一覧 API は存在する。
- 一方で、ユーザーが自分の連携済みメールアカウントを解除する API は未提供である。
- DDD 上の公開概念は `MailAccountConnection` だが、現行実装では `email_credentials` テーブルを backing store として利用している。

### 目的
- 認証済みユーザーが、自分自身のメール連携を 1 件単位で解除できるようにする。
- 解除後、その connection が一覧 API に出現せず、backend がその資格情報を利用しない状態にする。
- 現状は Gmail のみ対象だが、将来的な Outlook 等の追加に耐える HTTP 契約にする。

### 非スコープ
- Google 側の OAuth token revoke API 呼び出し
- 複数 connection の一括解除
- 解除済み connection の履歴表示
- 連携詳細単体取得 API
- バッチ設定との参照整合チェック

## 2. API 契約

### Endpoint
- Method: `DELETE`
- Path: `/api/v1/mail-account-connections/:connection_id`
- Auth: required
- Request body: なし

### Response 204
- body なし

### Error
- `400 invalid_request`
  - `connection_id` が不正
- `401 unauthorized`
  - JWT 不正または未認証
- `404 mail_account_connection_not_found`
  - 対象 connection が存在しない、または自分の所有ではない
- `500 internal_server_error`
  - DB 削除失敗など、解除処理の内部失敗

### 契約上の注意
- path parameter は resource 識別子として `connection_id` を使う。
- 成功時は `204 No Content` のため本文を返さない。
- `404` に ownership failure を含め、他ユーザーの connection 存在を漏らさない。
- token 値や digest はレスポンスに含めない。

## 3. 機能要件

- 認証済みユーザーのみ利用できること。
- 自分自身が所有する `MailAccountConnection` のみ解除できること。
- path parameter の `connection_id` で 1 件を特定できること。
- 解除後、対象 connection は一覧 API の結果に含まれないこと。
- 解除後、backend は対象 connection の資格情報を利用しないこと。
- レスポンスや構造化ログにアクセストークン / リフレッシュトークン / digest を出さないこと。

## 4. 削除方針

v1 は `MailAccountConnection` の active set を `email_credentials` row の存在で表現し、解除時は対象 row を削除する。

### 削除アルゴリズム
1. 認証コンテキストから `user_id` を取得する。
2. path parameter `connection_id` を parse する。
3. `id + user_id` で対象 credential を削除する。
4. 対象なしなら `not found` を返す。
5. 成功時は `204 No Content` を返す。

### 補足
- provider への問い合わせ、token 復号、Google revoke API 呼び出しは行わない。
- 将来 audit 要件が追加された場合も、公開契約は維持したまま logical revoke 実装へ差し替え可能である。

## 5. レイヤ設計

### Presentation
- `DELETE /api/v1/mail-account-connections/:connection_id` を追加する。
- handler は `(*Controller).Disconnect` とする。
- request body DTO は不要とする。
- `connection_id` parse 失敗は `400 invalid_request` を返す。

### Application
- `UseCaseInterface` に `Disconnect(ctx context.Context, userID uint, connectionID uint) error` を追加する。
- authorize / callback / list と同じ `emailcredential` usecase に置く。
- provider 呼び出しは行わず、ローカル state の除去だけを責務とする。

### Domain
- `ErrCredentialNotFound` を not found の判定に使う。
- 追加の read model は不要とする。

### Infrastructure
- Repository に `DeleteCredentialByIDAndUser(ctx context.Context, credentialID, userID uint) error` を追加する。
- 実装は `id + user_id` 条件で delete する。
- `RowsAffected == 0` は `ErrCredentialNotFound` を返す。

## 6. Provider 拡張方針

- HTTP path は provider 非依存の単体 resource path を維持する。
- 対象指定は `connection_id` で統一するため、Outlook 追加時も契約を変えない。
- v1 の解除は「backend がその connection を利用しない状態にする」ことに責務を限定する。
- provider 側 revoke を追加する場合でも、同一 endpoint の内部処理強化として扱う。

## 7. テスト観点

### Controller
- `204 No Content` 正常系
- `400 invalid_request`
- `401 unauthorized`
- `404 mail_account_connection_not_found`
- `500 internal_server_error`

### UseCase
- 所有 connection の解除成功
- 対象なしで `ErrCredentialNotFound`
- repository error の伝播

### Repository
- `id + user_id` 一致時のみ削除されること
- 他ユーザーの row は削除できないこと
- 対象なしで `ErrCredentialNotFound`

### Router
- `DELETE /api/v1/mail-account-connections/:connection_id` が登録される

## 8. 判断事項

- v1 は hard delete を採用する。
- v1 は Google 側 revoke を行わない。
- 解除対象の指定は provider 固有識別子ではなく connection ID とする。
