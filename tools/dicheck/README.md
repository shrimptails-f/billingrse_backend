# dicheck

このディレクトリには、`golangci-lint custom` で読む custom linter を置いています。

## ファイルごとの役割

- [diinterfacecheck.go](/home/dev/backend/tools/dicheck/diinterfacecheck.go)
  - `dig.Provide(func(...))` の引数をチェックします。
  - DI 配線では interface ではなく具象を受ける、というルールを見ています。
  - `defaultTargets` に「interface 名」と「使わせたい具象名」の対応を書いています。

- [newinterfacecheck.go](/home/dev/backend/tools/dicheck/newinterfacecheck.go)
  - `NewXxx` という名前のトップレベル関数をチェックします。
  - constructor 側では他 package の具象依存を直接受けず、interface を受ける、というルールを見ています。
  - 現状の例外は `*gorm.DB` と `*gin.Engine` です。

- `.custom-gcl.yml`
  - `golangci-lint custom` で custom binary を build するための設定です。
  - この package を plugin として読み込むために使います。

## 補足

- 実際の有効化設定は repo root の [/.golangci.yml](/home/dev/backend/.golangci.yml) にあります。
- 実行導線は repo root の [Taskfile.yml](/home/dev/backend/Taskfile.yml) の `task lint` を使います。
