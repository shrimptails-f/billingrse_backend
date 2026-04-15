# ADR 0001: DB 接続情報は専用 Secret client と起動時キャッシュで扱う

Status: Accepted  
Date: 2026-04-15

## Context
RDS 作成時のユーザー名やパスワードは AWS により自動生成された Secret に保存される。
この Secret 名は固定ではなく、環境ごとに変わりうる。

一方、現状の `internal/library/secret/client.go` は `defaultSecretName` に固定されており、
`internal/library/oswrapper/reader.go` は `MYSQL_USER` や `MYSQL_PASSWORD` などのキーを
その固定 Secret から都度取得する前提になっている。

このままだと次の問題がある。

- RDS 自動生成 Secret のランダムな名前を扱えない
- `GetEnv` のたびに SecretManager を参照しうるため、無駄な API 呼び出しが増える
- DB 用 Secret とアプリ共通 Secret が同一の取得経路に混在しており、責務が曖昧

また、運用方針として `APP` が `local` または `ci` 以外の環境では、
特定の機密項目は環境変数ではなく AWS Secrets Manager を正としたい。

## Decision
- `secret.New` は対象の Secret 名を受け取り、その Secret 全体を初期化時に 1 回だけ取得して保持する
- `defaultSecretName` はアプリ共通 Secret のデフォルト名として残し、
  DB 用 Secret は `DB_SECRET_NAME` で別途指定する
- サーバー起動時や seeder 起動時に、用途ごとに Secret client を分けて生成する
  - アプリ共通設定用の `appSecretClient`
  - DB 接続情報用の `dbSecretClient`
- `oswrapper.New` は `appSecretClient` と `dbSecretClient` を受け取り、
  `APP` が `local` または `ci` 以外の場合のみ DB 用 Secret を起動時に読み込んでキャッシュする
- `oswrapper` は DB 用 Secret の値を private field または内部 map に保持し、
  `GetEnv` で DB 関連キーが要求された場合はそのキャッシュを返す
- `GetEnv` の優先順位は次の通りとする
  - `APP` が `local` または `ci` の場合: 環境変数を返す
  - それ以外で DB 関連キーの場合: DB キャッシュを返す
  - それ以外で secret 対象キーの場合: アプリ共通 Secret から返す
  - 非 secret 対象キーの場合: 環境変数を返す
- RDS 自動生成 Secret のキー名は `oswrapper` 内で既存の設定キーへ正規化して扱う
  - 例: `username -> MYSQL_USER`
  - 例: `password -> MYSQL_PASSWORD`
  - 例: `host -> DB_HOST`
  - 例: `port -> DB_PORT`
  - 例: `dbname -> MYSQL_DATABASE`

## Consequences
- 非 `local`/`ci` 環境では、DB 接続情報の取得元を AWS Secrets Manager に統一できる
- RDS 自動生成 Secret のランダムな Secret 名を `DB_SECRET_NAME` で吸収できる
- DB 値は起動時にキャッシュされるため、リクエスト処理中や `GetEnv` 呼び出しごとの
  Secrets Manager API 呼び出しを避けられる
- アプリ共通 Secret と DB Secret の責務が分かれ、設定の見通しが良くなる
- DB 接続情報を使う既存コードは `MYSQL_USER` や `DB_HOST` などの既存キー名を
  そのまま使い続けられる
- `oswrapper.New` は DB Secret の事前読込に失敗しうるため、
  初期化エラーを返せる形へ見直す余地がある
