# MailAccountConnection Scenario Tests

`test/scenario/mail_account_connection` は、`MailAccountConnection` の主要ユースケースを HTTP 入口から確認するシナリオテスト群です。

## 前提

- `router`
- `auth middleware`
- `controller`
- `usecase`
- `repository`
- `MySQL`

上記は real で動かします。

- Google OAuth token exchange
- Gmail profile fetch

上記は helper で fake に差し替えています。  
目的は、Google SDK 自体ではなく、backend 側の state 管理・永続化・HTTP 契約・所有権制御を検証することです。

## 実行例

```bash
go test ./test/scenario/mail_account_connection -count=1
```

## シナリオ一覧

| テスト | ファイル | シナリオ | 主な観点 |
| --- | --- | --- | --- |
| `TestMailAccountConnection_FullFlowScenario` | `mail_account_connection_full_flow_scenario_test.go` | `authorize -> callback -> list -> disconnect -> list empty` の基本フロー | 認可 URL 発行、pending state 保存、callback 成功、token の暗号化保存、一覧レスポンス、`204 No Content`、解除後に一覧から消えること |
| `TestMailAccountConnectionRelinkScenario` | `mail_account_connection_relink_scenario_test.go` | 同一 Gmail アドレスの再連携 | 同一 connection で更新されること、row 数が増えないこと、ID が維持されること、暗号化 token 列が更新されること |
| `TestMailAccountConnectionMultipleGmailAccountsScenario` | `mail_account_connection_multiple_accounts_scenario_test.go` | 同一ユーザーで複数 Gmail を連携 | 別 Gmail は別 connection として作成されること、row 数が 2 件になること、一覧に両方の `account_identifier` が出ること |
| `TestMailAccountConnectionScenario_CallbackMismatchedState` | `mail_account_connection_error_scenario_test.go` | state 不一致で callback | `409 oauth_state_mismatch`、credential 未作成、異常時に保存が走らないこと |
| `TestMailAccountConnectionScenario_CallbackExpiredState` | `mail_account_connection_error_scenario_test.go` | state 期限切れで callback | `409 oauth_state_expired`、credential 未作成、期限切れ state を受け付けないこと |
| `TestMailAccountConnectionScenario_DeleteOtherUsersConnectionKeepsOriginalCredential` | `mail_account_connection_error_scenario_test.go` | 他ユーザーの connection を削除しようとする | `404 mail_account_connection_not_found`、所有権を跨いだ削除ができないこと、元の credential が残ること |

## helper の責務

| ファイル | 役割 |
| --- | --- |
| `mail_account_connection_scenario_helpers_test.go` | テスト DB 構築、router 初期化、JWT 発行、fake Google 依存、HTTP request helper、DB 確認 helper を提供する |
