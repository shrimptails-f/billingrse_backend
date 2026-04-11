# ADR 0006: ユビキタス言語図では永続的な直接関係のみを線で表す

Status: Accepted  
Date: 2026-04-11

## Context
`docs/ddd/README.md` と `docs/ddd/ubiquitous-language/README.md` のクラス図では、
関係線が「実行時に使う」「概念上は関連する」という関係まで広がり、
永続化上の直接参照と区別しづらくなっていた。

特に次の 2 つは、図に直接線を残すと実際のモデルより強い関連を示してしまう。

- `ManualMailWorkflowHistory` と `MailAccountConnection`
  - 手動実行履歴のレコードは `mail_account_connection_id` を保持せず、
    削除耐性のため provider などのスナップショット値を保持する
  - そのため、履歴からメールアカウント連携へ永続的に参照している関係ではない
- `BatchSetting` と `MailAccountConnection`
  - `BatchSetting` はユーザーが所有し、`FetchCondition` を内包する独立集約として整理している
  - 現在のユビキタス言語図では、この関係を直接線で強調すると
    構造上の必須参照であるかのように読めてしまう

一方で、手動実行と履歴の関係は `ManualMailFetch -> ManualMailWorkflowHistory` として
「実行結果を保持する」関係で表した方が、実際の責務と永続化の形に近い。

## Decision
- ユビキタス言語図および DDD のクラス図では、永続化上または集約境界上の
  直接的で安定した関係のみを線で表す
- 実行時の入力、処理手順上の関係、スナップショット保存で代替されている関係は、
  原則として直接線で表さない
- `ManualMailWorkflowHistory` と `MailAccountConnection` の直接線は削除する
- `BatchSetting` と `MailAccountConnection` の直接線は削除する
- `ManualMailFetch` と `ManualMailWorkflowHistory` の関係は、
  `実行結果を保持` として表現する

## Consequences
- 図から `MailAccountConnection` への線が減るが、これは関連がゼロという意味ではなく、
  永続的な直接参照としては扱わない、という意味になる
- `ManualMailWorkflowHistory` について、読者は
  「接続情報を FK 参照する履歴」ではなく
  「実行時情報のスナップショットを持つ監査用集約」と理解しやすくなる
- `BatchSetting` について、`User` 所有と `FetchCondition` 内包が主な構造であることを
  図で優先して表現できる
- 今後、これらのモデルが実際に永続的な直接参照を持つように変わった場合は、
  図の線を復活させるかを別途見直す
