# 集約境界（概念レベル）

本ドキュメントは、ドメインモデルの集約境界を概念レベルで整理する。

参照:
- `docs/ddd/ubiquitous-language.md`
- `docs/ddd/invariants.md`
- `docs/ddd/domain-model.md`

## 集約一覧

### ユーザー集約
- ルート: ユーザー（User）
- 説明: データ分離の単位

### メールサービス集約
- ルート: メールサービス（MailService）
- 説明: 参照データの集約

### メールアカウント連携集約
- ルート: メールアカウント連携（MailAccountConnection）
- 含む: メール取得バッチ（MailFetchBatch）
- バッチ設定: 取得条件 / 実行スケジュール（いずれも連携に紐づく）
- 参照: ユーザー / メールサービス

### メール集約
- ルート: メール（Email）
- 含む: メール解析結果（ParsedEmail）
- 参照: ユーザー

### 請求集約
- ルート: 請求（Billing）
- 参照: 支払先（Vendor）/ メール（Email）/ 支払いタイプ（PaymentType）

### 支払先集約
- ルート: 支払先（Vendor）
- 説明: 参照データの集約

## 補足

- メール取得（MailFetch）は概念であり、集約として扱わない
- 不変条件の詳細は `docs/ddd/invariants.md` を正とする
