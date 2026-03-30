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
  - 現状は、interface 越しの呼び出しは「ローカル代入元から concrete 実装が分かる」か、「current package と direct import から実装候補を列挙できる」範囲だけ追います。
  - repository wrapper の実体解決までは追いません。
  - 他 package 追跡は current package と同じ workspace 配下にある import 先に限定しています。

- [nplusonecheck_test.go](/home/dev/backend/tools/nplusonecheck/nplusonecheck_test.go)
  - `analysistest` を使った最小テストです。
  - `testdata/src/<package>` 配下の各サンプルで、期待する診断位置に `// want "possible N\+1 query inside loop"` を書いて検証します。
  - `analysistest.Run(...)` は複数 package をまとめて実行していますが、`// want` の照合は各 package / 各行ごとに行われます。失敗時は該当する `testdata` のファイルと行が出るため、どのケースが崩れたかは判別できます。
  - 一方で `go test -v` 上は `TestAnalyzer` 1件の表示なので、各ケースが個別に `PASS` した形では見えません。個別表示が欲しい場合は package ごとに `t.Run(...)` で subtest 化します。

- [.custom-gcl.yml](/home/dev/backend/tools/nplusonecheck/.custom-gcl.yml)
  - `golangci-lint custom` で custom binary を build するための設定です。
  - 現在は既存の `dicheck` と `nplusonecheck` を同じ custom binary に載せています。

## 補足

- 実際の有効化設定は repo root の [/.golangci.yml](/home/dev/backend/.golangci.yml) にあります。
- 実行導線は repo root の [Taskfile.yml](/home/dev/backend/Taskfile.yml) の `task lint` を使います。

## 現状の対応範囲

この linter は、まず `for` / `range` を起点に見つけます。
そのループ本体配下の `CallExpr` をたどり、最終的に `database/sql` / `sqlx` / `gorm` の既知のクエリ実行メソッドへ到達するかを判定します。
つまり実装は、大きく「ループ検知」と「呼び出し解決」に分かれます。

### 対応済み

| パターン | 概要 | 実装箇所 |
| --- | --- | --- |
| 同一 package の helper 関数 | 例: `for ... { findUser(db, id) }` の先で `db.QueryRow(...)` や `db.First(...)` を呼ぶ形です。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go), [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |
| receiver method | 例: `for ... { svc.loadUser(id) }` や `repo.FindByID(id)` の先で query する形です。 | [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |
| 他 package 呼び出し | current package と同じ workspace 配下にある import 先 package だけを追跡します。 | [package_loader.go](/home/dev/backend/tools/nplusonecheck/package_loader.go), [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |
| interface 経由の呼び出し | 例: `var repo UserRepository = &sqlRepo{...}; repo.FindByID(id)` や `uc.repository.FindByID(ctx, id)`。receiver の代入元から concrete 実装が分かる場合はその実装を使い、そうでなければ current package と direct import から実装候補を列挙して、query 到達候補があれば保守的に検知します。 | [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go), [package_loader.go](/home/dev/backend/tools/nplusonecheck/package_loader.go), [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| 構造体 field の奥にある DB 呼び出し | 例: `s.userRepo.db.First(...)`。最終的な receiver 式の型が `*gorm.DB` / `*sql.DB` / `*sql.Tx` に解決できれば検知します。 | [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| query builder を外で作って中で execute | 例: `q := db.Where(...); for ... { q.First(...) }`。変数伝播そのものを追うのではなく、実行点の receiver 型で判定します。 | [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| transaction / scoped DB の派生 | 例: `tx := db.WithContext(ctx)` や `scoped := db.Session(...)` のあとに `tx.First(...)` / `scoped.Find(...)` する形です。 | [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| closure / 無名関数の即時実行 | 例: `for ... { func(){ db.First(...) }() }`。`go` / `defer` の呼び出し式配下にある query も、AST 上たどれる範囲では検知対象です。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go), [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |
| nested loop | 内側ループで query していれば通常どおり警告します。`N+1` と `N×M` の区別はしていません。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go) |
| キャッシュ miss 時だけ query | 例: `if _, ok := cache[id]; !ok { db.First(...) }`。条件分岐の精密解析はしていないため、query 到達経路があれば保守的に警告します。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go), [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |

### 未対応または限定的

| パターン | 概要 | 実装箇所 |
| --- | --- | --- |
| loop を隠した高階関数 | 例: `lo.ForEach`, `slices` 系、独自 `Each`。起点が `for` / `range` の AST に限定されるため、現状は対象外です。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go) |
| ORM の関連読み込み | 例: `Association(...).Find(...)` や lazy load 相当です。現在の query matcher は `gorm.DB` の一部メソッドだけを対象にしています。 | [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| hook / callback | 例: `AfterFind`, serializer, scan hook などで暗黙に query が走る形です。明示的な call graph と receiver 型の追跡だけでは扱えません。 | [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go), [query_matcher.go](/home/dev/backend/tools/nplusonecheck/query_matcher.go) |
| closure / 無名関数の未実行定義 | 例: `fn := func() { db.First(...) }`。実行経路に乗ることが AST から直ちに分からないため、現状は中身を見ません。 | [loop_detector.go](/home/dev/backend/tools/nplusonecheck/loop_detector.go), [local_call_graph.go](/home/dev/backend/tools/nplusonecheck/local_call_graph.go) |

現状は PoC なので、検知漏れを減らすよりも、まずは分かりやすい query 実行点を軽量に拾う方を優先しています。
また `gorm` については「実行メソッド」を query とみなしているため、`Where(...)` や `Session(...)` のような builder / scope 生成だけでは報告しません。
また interface method は保守的に扱っており、実装候補が複数あっても、その中に query 到達候補があれば警告します。
