# API 仕様一覧

`docs/spec` 配下の API 設計書一覧です。

| API名 | HTTPメソッド | エンドポイント | 設計書 | 説明 |
| --- | --- | --- | --- | --- |
| Billing 一覧 API | `GET` | `/api/v1/billings` | [BillingList.md](./BillingList.md) | 認証済みユーザー自身の請求一覧を、検索中心で取得する。 |
| Gmail OAuth 認可 URL 発行 API | `POST` | `/api/v1/mail-account-connections/gmail/authorize` | [MailAccountConnection.md](./MailAccountConnection.md) | 認証済みユーザー向けに Gmail OAuth の認可 URL と有効期限を発行する。 |
| Gmail OAuth コールバック受付 API | `POST` | `/api/v1/mail-account-connections/gmail/callback` | [MailAccountConnection.md](./MailAccountConnection.md) | frontend から受け取った `code` と `state` を検証し、MailAccountConnection を作成または再連携する。 |
| MailAccountConnection 一覧 API | `GET` | `/api/v1/mail-account-connections` | [MailAccountConnectionList.md](./MailAccountConnectionList.md) | 認証済みユーザー自身のメール連携一覧を返す。provider へのリアルタイム確認は行わない。 |
| MailAccountConnection 連携解除 API | `DELETE` | `/api/v1/mail-account-connections/:connection_id` | [MailAccountConnectionDisconnect.md](./MailAccountConnectionDisconnect.md) | 指定した `connection_id` のメール連携を、自分の所有範囲に限定して解除する。 |
| 手動メール取得履歴一覧 API | `GET` | `/api/v1/manual-mail-workflows` | [ManualMailWorkflowHistoryList.md](./ManualMailWorkflowHistoryList.md) | 認証済みユーザー自身の手動メール取得履歴を、stage 件数と failure 明細付きで一覧返却する。 |
