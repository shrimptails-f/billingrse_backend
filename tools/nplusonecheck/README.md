# nplusonecheck

このディレクトリには、`golangci-lint custom` で読む N+1 検知用 custom linter を置いています。

## ファイルごとの役割

- [plugin.go](/home/dev/backend/tools/nplusonecheck/plugin.go)
  - plugin 登録と analyzer の入口です。

- [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go)
  - `database/sql` / `sqlx` / `gorm` のどのメソッドをクエリ実行として扱うかを定義します。

- [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go)
  - current package と import 先 package の helper 関数や method を再帰的にたどります。

- [package_loader.go](/home/dev/backend/tools/nplusonecheck/package_loader.go)
  - import 先 package を source 付きで読み込み、workspace 配下の package だけを追跡対象にします。

- [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go)
  - `for` / `range` の中で、クエリ実行につながる呼び出しを検知して report します。
  - メッセージは `possible N+1 query inside loop` です。

- 補足
  - 現状は PoC なので、interface 越しの呼び出し、repository wrapper の実体解決までは追いません。
  - 他 package 追跡は current package と同じ workspace 配下にある import 先に限定しています。

- [nplusonecheck_test.go](/home/dev/backend/tools/nplusonecheck/nplusonecheck_test.go)
  - `analysistest` を使った最小テストです。

- [.custom-gcl.yml](/home/dev/backend/tools/nplusonecheck/.custom-gcl.yml)
  - `golangci-lint custom` で custom binary を build するための設定です。
  - 現在は既存の `dicheck` と `nplusonecheck` を同じ custom binary に載せています。

## 補足

- 実際の有効化設定は repo root の [/.golangci.yml](/home/dev/backend/.golangci.yml) にあります。
- 実行導線は repo root の [Taskfile.yml](/home/dev/backend/Taskfile.yml) の `task lint` を使います。
