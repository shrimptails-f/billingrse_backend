# ADR 0001: DB 接続情報は専用 Secret client と起動時キャッシュで扱う

Status: Accepted  
Date: 2026-04-15

## Context
RDS 作成時のユーザー名やパスワードは AWS により自動生成された Secret に保存される。
この Secret 名は固定ではなく、環境ごとに変わりうる。
また、RDS 自動生成 Secret の JSON キーは `username` / `password` であり、
アプリケーション側で参照している `MYSQL_USER` / `MYSQL_PASSWORD` とは一致しない。

一方、アプリケーションでは DB 接続設定の一部として `MYSQL_USER` / `MYSQL_PASSWORD` /
`DB_HOST` / `DB_PORT` を参照しており、アプリ共通 Secret と DB 用 Secret の責務も分けたい。

このままだと次の問題がある。

- app / DB の Secret 名が環境ごとに変わる場合をコード固定値で扱えない
- `GetEnv` のたびに SecretManager を参照しうるため、無駄な API 呼び出しが増える
- DB 用 Secret とアプリ共通 Secret が同一の取得経路に混在しており、責務が曖昧
- RDS 自動生成 Secret のキー名とアプリ側の env 名が一致しない

また、運用方針として `APP` が `local` または `ci` 以外の環境では、
特定の機密項目は環境変数ではなく AWS Secrets Manager を正としたい。

## Decision
- `secret.New` は対象の Secret 名を受け取り、その Secret 全体を初期化時に 1 回だけ取得して保持する
- Secret 名はコードに埋め込まず、起動時に環境変数から受け取る
  - アプリ共通設定用 Secret: `APP_SECRET_NAME`
  - DB 接続情報用 Secret: `DB_SECRET_NAME`
- サーバー起動時や seeder 起動時に、用途ごとに Secret client を分けて生成する
  - アプリ共通設定用の `appSecretClient`
  - DB 接続情報用の `dbSecretClient`
- `oswrapper.New` は `appSecretClient` と `dbSecretClient` を受け取り、
  `APP` が `local` または `ci` 以外の場合のみ DB 用 Secret を起動時に読み込んでキャッシュする
- `oswrapper` は DB 用 Secret の値を private field または内部 map に保持し、
  `GetEnv` で DB 関連キーが要求された場合はそのキャッシュを返す
- DB 用 Secret からは RDS 自動生成 Secret のキー名に合わせて次を読み取る
  - `username` -> `MYSQL_USER`
  - `password` -> `MYSQL_PASSWORD`
- `DB_HOST` と `DB_PORT` は DB 用 Secret ではなくアプリ共通 Secret から読み取る
- `GetEnv` の優先順位は次の通りとする
  - `APP` が `local` または `ci` の場合: 環境変数を返す
  - それ以外で `MYSQL_USER` / `MYSQL_PASSWORD` の場合: DB キャッシュを返す
  - それ以外で secret 対象キーの場合: アプリ共通 Secret から返す
  - 非 secret 対象キーの場合: 環境変数を返す

## Consequences
- 非 `local`/`ci` 環境では、DB 接続情報の取得元を AWS Secrets Manager に統一できる
- app / DB の Secret 名の環境差分を `APP_SECRET_NAME` / `DB_SECRET_NAME` で吸収できる
- DB 値は起動時にキャッシュされるため、リクエスト処理中や `GetEnv` 呼び出しごとの
  Secrets Manager API 呼び出しを避けられる
- RDS 自動生成 Secret の `username` / `password` をアプリ側の `MYSQL_USER` /
  `MYSQL_PASSWORD` に変換して扱える
- `DB_HOST` / `DB_PORT` は app Secret 側で一元管理できる
- アプリ共通 Secret と DB Secret の責務が分かれ、設定の見通しが良くなる
- `oswrapper.New` は DB Secret の事前読込に失敗しうるため、
  初期化エラーを返せる形へ見直す余地がある
