# ADR 0002: メール取得はワークフロー扱い、ParsedEmail の多件化とモデル更新

Status: Accepted  
Date: 2026-01-30

## Context
ADR 0001 ではメール取得（MailFetch/ManualMailFetch/MailFetchBatch）をドメインモデルに含め、
ParsedEmail を「1メールにつき1件」としていた。しかし現時点では取得の手段・タイミングに
業務ルールの差がなく、モデルに含める必要性が低い。また、解析の再実行などを考慮すると
ParsedEmail は複数件になり得る。

## Decision
- メール取得（MailFetch/ManualMailFetch/MailFetchBatch）はドメインモデルから除外し、
  アプリケーション層のワークフローとして扱う。用語としてはユビキタス言語に残す。
- ParsedEmail は 1メールにつき 0件以上（複数件）を許容する。
- ParsedEmail のモデルは請求生成に必要な推定項目（支払先名、請求番号、金額、通貨、請求日、支払いタイプ）
  と抽出日時を保持する。

## Consequences
- ドメインモデル図から MailFetch 系のエンティティ/概念が削除される。
- 不変条件の「1メールにつきメール解析結果は1件のみ」は廃止される。
- 解析結果の履歴・再解析を表現できるようになる。
