# 環境構築手順
## openai APIキーを発行
1. https://openai.com/ja-JP/api/ にアクセスしてアカウント登録
2. https://platform.openai.com/api-keys にアクセスしてAPIキーを発行
3. 発行したAPIキーを控えておく

## 注意事項
- OpenAI APIの利用には料金が発生します
- APIキーは安全に管理してください

## Google OAuth2設定
1. [Google Cloud Console](https://console.cloud.google.com/)にアクセス
2. https://console.developers.google.com/apis/library にアクセスし、「Gmail API」で検索→Gmail-apiを有効化する。
4. 「https://console.cloud.google.com/apis/credentials にアクセスし、認証情報を作成→OAuthクライアントIDを選択する。
5. アプリケーションの種類として「ウェブアプリケーション」を選択
6. 承認済みのリダイレクトURIに `http://localhost:5555/Callback` を追加
7. 秘密鍵をJSONでダウンロード あとで使います。

## ソースをクローン
```bash
git clone https://github.com/shrimptails-f/billingrse.git
```
## .envをコピー
```bash
cp .devcontainer/.env.sample .devcontainer/.env
```
## 環境変数の設定

アプリケーションの実行には以下の環境変数が必要です。`.env` ファイルに設定してください。

### メールアカウント認証情報暗号化関連
- `EMAIL_TOKEN_KEY_V1`: メールトークン暗号化用のマスターキー（32バイト以上の文字列を推奨）
- `EMAIL_TOKEN_SALT`: Digest 生成用のソルト（ランダムな文字列を推奨）
- `EMAIL_GMAIL_CLIENT_ID`: Google OAuth2設定で取得した クライアントID（必須）
- `EMAIL_GMAIL_CLIENT_SECRET`: Google OAuth2設定で取得した クライアントシークレット（必須）
- Gmail OAuth state の有効期限と refresh skew はコード定数（`internal/common/const.go`）で管理する

### API レート制御
- `REDIS_HOST`: Redis ホスト名（例: `redis`)
- `REDIS_PORT`: Redis ポート（例: `6379`)
- `REDIS_PASSWORD`: Redis パスワード（例: `redis_local_password`、未設定可）
- `REDIS_DB`: Redis DB 番号（例: `0`)
- レート制御の既定値はコード定数（`internal/common/const.go`）で管理する

### OpenAI API
OPENAI_API_KEY=生成したOpenAiのAPIキーを記載

## VsCodeでプロジェクトフォルダーを開く
## Reopen in Containerを押下
もし表示されない場合は Ctrl Shift P→Reopen in containerと入力して実行でもおｋ

## テーブル作成
```bash
task migration-create
```

# 環境構築完了です！！
お疲れ様でした。
