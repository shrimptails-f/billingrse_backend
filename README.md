![Version](https://img.shields.io/badge/Version-1.0.0-green) [![Go CI](https://github.com/shrimptails-f/billingrse/actions/workflows/go_ci.yml/badge.svg)](https://github.com/shrimptails-f/billingrse/actions/workflows/go_ci.yml)

# billingrse - AIで営業メールを構造化
ITエンジニア向けの営業メールを Gmail から取得し、AI(OpenAI ChatGPT) で解析、レスポンスを構造化して MySQL に蓄積。RESTL で検索・集計できる Go 製バックエンドです。

## プロジェクト概要
- メールサービス連携（MailAccountConnection）を管理し、メール取得（手動/バッチ）で Email を取得・保存
- Email を AI 解析して ParsedEmail を生成し、請求成立判定で Billing を確定（Vendor/金額/日付など）
- REST API と SQL で Email/ParsedEmail/Billing を検索・集計
- JWT 認証とメールサービス連携により、会員ごとにデータを分離

## 集計イメージ
![image](https://github.com/user-attachments/assets/fee1cd7c-b4c8-428c-806e-2dbfb4eb51a5)

## クイックスタート
TODO:

## アーキテクチャ / 技術スタック
- アーキテクチャ:
  - Clean Architecture（presentation / application / domain / infrastructure）
- 言語/フレームワーク: 
  - Go 1.24
  - Gin
  - Gorm
  - go.uber.org/dig（DI）
- 外部API: Gmail（OAuth2）, OpenAI API クライアント
- 認証: JWT
- 非同期パイプライン: Gmail 取得 → OpenAI 解析 → emailstore 保存（レートリミット/ログ付き）
- DB/NoSQL: MySQL 8.0, Redis（APIのレート制御）
- 開発環境: Docker/DevContainer, Taskfile, Air（ホットリロード）
- 主要ライブラリ:
  - gin-gonic/gin
  - gorm / gorm.io/driver/mysql
  - go.uber.org/zap, go.uber.org/dig
  - golang-jwt/jwt
  - google.golang.org/api (Gmail)
  - github.com/openai/openai-go (ChatGPT)
  - redis/go-redis/v9
  - stretchr/testify, golang/mock

## ディレクトリ構成
### 詳細
アーキテクチャ詳細は[こちら](./docs/architecture.md)を参照してください。

## 開発・品質
- テスト: `task test`（`go test -race -cover` + `coverage.html` 出力、`t.Parallel()` で並列安全性検証）
- CI: GitHub Actions [Go CI](.github/workflows/go_ci.yml) がビルド・`go vet`・`go test -race` を実行
- マイグレーション/シード: [docs/migration.md](./docs/migration.md) を参照
- SQL 例: [docs/query.md](./docs/query.md)

## その他メモ
- codex のブラウザ認証でリダイレクトされない場合  
  Ctrl Shift P → run task → codex callback access を押下 → ブラウザのアドレスバーの `http://localhost:1455/auth/callback?...` をまるまるコピペして実行するとログインできます。
