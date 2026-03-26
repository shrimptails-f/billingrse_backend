# API 仕様一覧

`docs/spec` 配下の API 設計書一覧です。

| API名 | HTTPメソッド | エンドポイント | 説明 |
| --- | --- | --- | --- |
| [Billing 一覧 API](./BillingList.md) | `GET` | `/api/v1/billings` | 認証済みユーザー自身の請求一覧を、検索中心で取得する。 |
| [Billing Monthly Trend API](./BillingMonthlyTrend.md) | `GET` | `/api/v1/billings/summary/monthly-trend` | 認証済みユーザー自身の請求を、通貨別の直近 12 ヶ月 zero-fill 推移として取得する。 |
| [Billing Month Detail API](./BillingMonthDetail.md) | `GET` | `/api/v1/billings/summary/monthly-detail/:year_month` | 認証済みユーザー自身の請求を、指定月の支払先別内訳付き詳細として取得する。 |
| [Gmail OAuth 認可 URL 発行 API](./MailAccountConnection.md) | `POST` | `/api/v1/mail-account-connections/gmail/authorize` | 認証済みユーザー向けに Gmail OAuth の認可 URL と有効期限を発行する。 |
| [Gmail OAuth コールバック受付 API](./MailAccountConnection.md) | `POST` | `/api/v1/mail-account-connections/gmail/callback` | frontend から受け取った `code` と `state` を検証し、MailAccountConnection を作成または再連携する。 |
| [MailAccountConnection 一覧 API](./MailAccountConnectionList.md) | `GET` | `/api/v1/mail-account-connections` | 認証済みユーザー自身のメール連携一覧を返す。provider へのリアルタイム確認は行わない。 |
| [MailAccountConnection 連携解除 API](./MailAccountConnectionDisconnect.md) | `DELETE` | `/api/v1/mail-account-connections/:connection_id` | 指定した `connection_id` のメール連携を、自分の所有範囲に限定して解除する。 |
| [手動メール取得開始 API](./manualmailworkflow/requirementsDefinition.md) | `POST` | `/api/v1/manual-mail-workflows` | 手動メール取得 workflow を受け付け、処理完了を待たずに `202 Accepted` を返す。 |
| [手動メール取得履歴一覧 API](./ManualMailWorkflowHistoryList.md) | `GET` | `/api/v1/manual-mail-workflows` | 認証済みユーザー自身の手動メール取得履歴を、stage 件数と failure 明細付きで一覧返却する。 |
