# internal/mailfetch 要件定義

## 背景

- `docs/ddd/ubiquitous-language/mail-integration-fetch.md` では、`ManualMailFetch` は「手動トリガーで即時実行される Email 取得」であり、Email の意味解釈を持たない。
- `docs/spec/manualmailworkflow/設計.md` では、手動メール取得全体を `manualmailworkflow -> mailfetch -> emailanalysis -> billing` の stage に分割する方針が定義されている。
- 現在のリポジトリでは `internal/mailaccountconnection` と Gmail クライアント群に加え、raw Email の取得・保存を担う `internal/mailfetch` が実装済みである。
- `internal/common/domain.FetchedEmailDTO` は既に存在し、Gmail クライアントもこの DTO を返す。
- `Email` は metadata 保存に責務を限定し、本文は永続化しない方針とする。
- 後段の `emailanalysis` が本文を必要とするため、同一 workflow 内で fetch 結果を `created_emails` として引き継ぐ前提で整理する。
- Email の一意性は `user_id + external_message_id` を採用し、provider / account source は metadata として保持する。

## 目的

- `internal/mailfetch` を、選択された `MailAccountConnection` から raw Email を取得し、idempotent に保存する stage package として定義する。
- `mailfetch` の責務を「取得条件の検証」「接続情報の解決」「provider からの取得」「Email 保存」「取得結果サマリ返却」に限定する。
- 後段の `emailanalysis` と `billing` が安定して利用できるよう、保存対象・一意性・戻り値を先に確定する。

## スコープ

- `internal/mailfetch` の application / domain / infrastructure の責務分割
- `mailfetch` が受ける command と返す result
- `MailAccountConnection` / Gmail / Email 永続化との接続点
- Gmail v1 実装の取得条件と filtering 方針
- Email の保存キーと保存対象項目
- エラー分類と partial success の扱い
- DI / テスト観点

## 非スコープ

- `manualmailworkflow` 全体の orchestration
- `emailanalysis` の prompt / AI 実行 / `ParsedEmail` 保存
- `billing` の `VendorResolution` / `BillingEligibility` / `Billing` 保存
- 定期実行の `MailFetchBatch`
- 添付ファイル保存
- raw MIME 全文保存
- Outlook など Gmail 以外の具体実装

## 想定ユースケース

1. 認証済みユーザーが、連携済みメールアカウント 1 件を指定して手動取得を開始する。
2. backend は `connection_id` が `user_id` の所有物であり、取得可能状態であることを確認する。
3. backend は `FetchCondition` を検証する。
4. backend は provider に応じた `MailFetcher` を選択する。
5. `MailFetcher` は対象ラベルと期間に一致するメールを取得する。
6. backend は取得した raw メールを idempotent に保存する。
7. backend は `created_email_ids` と `existing_email_ids` を分けて返す。
8. backend は新規保存できたメール本文を含む `created_emails` も返す。
9. 上位の `manualmailworkflow` は、`created_emails` を後段の `emailanalysis` に渡す。

## 入力契約

- `user_id`
  - 実行主体のユーザー
- `connection_id`
  - 対象 `MailAccountConnection`
- `fetch_condition`
  - `label_name`
  - `since`
  - `until`

補足:
- DDD で要求されている取得条件は「期間 + ラベル」であるため、v1 は `label_name + since + until` を必須とする。
- 1 回の実行で対象にする connection は 1 件のみとする。
- v1 は 1 ラベルのみを受ける。

## 出力契約

- `provider`
- `account_identifier`
- `matched_message_count`
- `created_email_ids`
- `created_emails`
  - 新規保存できた message の一時 payload
  - 各要素は `email_id`, `external_message_id`, `subject`, `from`, `to`, `date`, `body` を持つ
- `existing_email_ids`
- `failures`
  - provider 取得や保存の途中で失敗した message 単位の失敗要約
  - save 失敗は 20 件 chunk の insert 失敗に起因して、当該 chunk 内の message へ展開されることがある

補足:
- `account_identifier` は取得対象メールの `To` ではなく、「どのメールアカウントから取得したか」を表す source 識別子である。
  - Gmail では連携済み Gmail アドレスが入る。
- `created_emails` は `created_email_ids` と同じ集合を入力順で表す一時 payload とする。
- 既存 Email の ID も返すが、下流の通常経路で使うのは `created_emails` と `created_email_ids` を正とする。
- 同一条件での再実行時に既存 Email を再解析し続けるのを避けるためである。

## 機能要件

- `mailfetch` は `FetchCondition` の構文と必須項目を検証すること。
- `mailfetch` は `connection_id` と `user_id` の所有関係を確認すること。
- 取得対象 connection が失効・無効・未対応 provider の場合は実行を開始しないこと。
- provider の選択は `MailFetcherFactory` に集約すること。
- Gmail v1 は既存の Gmail クライアントを再利用し、ラベル検索と message detail 取得を行うこと。
- Gmail v1 は label が存在しない場合、top-level error として扱うこと。
- provider 取得結果は `internal/common/domain.FetchedEmailDTO` に正規化すること。
- raw Email の保存では件名・送信元・宛先・受信日時などの metadata を保存すること。
- 本文は永続化対象に含めないこと。
- `mailfetch` の結果には、新規保存できた message の本文を `created_emails` として含めること。
- Email 保存は idempotent であること。
- Email 保存は 20 件単位で batch 化され、短い DB transaction で処理されること。
- idempotency key は `user_id + external_message_id` とすること。
- provider の一覧取得成功後、一部 message detail または保存で失敗しても、残りを継続できること。
- batch insert が失敗した chunk は、その chunk 内の message を `save` failure として扱い、後続 chunk の保存は継続すること。
- `mailfetch` は `emailanalysis` を直接起動しないこと。

## 非機能要件

- アクセストークン、リフレッシュトークン、auth code はログや result に含めない。
- Gmail API 呼び出しの retry / rate limit は既存 `internal/library/gmail` に委譲する。
- `context.Context` を受け取り、外部 I/O と DB へ伝播する。
- provider 依存の実装差分は infrastructure adapter に閉じ込める。
- 結果サマリだけで「作成」「既存」「失敗」を切り分けられること。
- save chunk 失敗時は、調査用に `user_id`, `provider`, `chunk_start_index`, `chunk_size`, `chunk_external_message_ids` を構造化ログへ出せること。

## 設計前提として確定したい論点

### 1. Email 本文は保存しない

- `mailfetch` が永続化する `Email` は metadata のみに限定する。
- 本文は `FetchedEmailDTO` の一時データとして扱い、保存対象には含めない。
- `emailanalysis` が本文を必要とする場合は、同一 workflow 内で `created_emails` として引き継ぐ。
- `provider + account_identifier + external_message_id` による再取得は、通常経路ではなく補助手段として扱う。

### 2. Email の一意性は `user_id + external_message_id` にする

- v1 の保存キーは `user_id + external_message_id` を採用する。
- `account_identifier` はメールの `To` ではなく、取得元 mailbox の識別子である。
  - Gmail では連携済み Gmail アドレスを使う。
- provider / `account_identifier` は再取得や監査用 metadata として `emails` テーブルへ保持する。
- `connection_id` は `emails` テーブルへ保持しない。

### 3. Gmail の期間上限は adapter 側で補完する

- 現行 `internal/library/gmail.Client.GetMessagesByLabelName` は `label + startDate` のみを受ける。
- そのため v1 Gmail adapter は `since` を Gmail query に使い、`until` は detail 取得後の `ReceivedAt` で後段 filter する。
- 将来 Gmail query builder を拡張した時点で adapter 内部実装を差し替える。

## 成功条件

- `internal/mailfetch` の責務が「取得と保存」に限定されていることが文書で共有できる。
- `MailAccountConnection` と Email 保存の接続方法が明確であること。
- Gmail v1 の実装方針と provider 拡張点が明確であること。
- Email が metadata 保存に限定されることと、一意性ルールが明確であること。
- `manualmailworkflow` が下流へ渡すべき本文付き payload を `created_emails` として明確にできること。

## TODO:
メール取得し保存に成功したメールの `created_emails` を返却しメール解析にかける想定だが、
解析処理済みのメールと解析に失敗したメールを再度 `created_emails` の返却対象にするのか要検討
解析成功していたら、基本は再解析はいらないはず。プロンプトが変わったから再解析したいとかなければ。
失敗は原因しだいかな?
