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

### エージェントトークン暗号化関連
- `AGENT_TOKEN_KEY_V1`: エージェントトークン暗号化用のマスターキー（32バイト以上の文字列を推奨）
- `AGENT_TOKEN_SALT`: ダイジェスト生成用のソルト（ランダムな文字列を推奨）

### メールアカウント認証情報暗号化関連
- `EMAIL_TOKEN_KEY_V1`: メールトークン暗号化用のマスターキー（32バイト以上の文字列を推奨）
- `EMAIL_TOKEN_SALT`: Digest 生成用のソルト（ランダムな文字列を推奨）
- `EMAIL_GMAIL_CLIENT_ID`: Gmail OAuth クライアントID（必須）
- `EMAIL_GMAIL_CLIENT_SECRET`: Gmail OAuth クライアントシークレット（必須）
- `EMAIL_GMAIL_REDIRECT_URL`: OAuth コールバック URL（例: `http://localhost:3000/account_links/callback`）
- `EMAIL_GMAIL_STATE_TTL_SECONDS`: OAuth state の有効期限（秒、デフォルト: 600）

### API レート制御
- `REDIS_HOST`: Redis ホスト名（例: `redis`)
- `REDIS_PORT`: Redis ポート（例: `6379`)
- `REDIS_PASSWORD`: Redis パスワード（例: `redis_local_password`、未設定可）
- `REDIS_DB`: Redis DB 番号（例: `0`)
- `REDIS_RATE_LIMIT_RPS`: 基本 RPS（省略可、デフォルト: 各 API の設定値）
- `REDIS_RATE_LIMIT_WINDOW_CONFIG`: ウィンドウ設定（例: `1:10,10:50,60:300`、省略可）
- `GMAIL_API_REQUESTS_PER_SECOND`: Redis RPS 設定が無い場合の Gmail デフォルト RPS（デフォルト: 10）
- `OPENAI_API_REQUESTS_PER_SECOND`: Redis RPS 設定が無い場合の OpenAI デフォルト RPS（デフォルト: 10）

**重要**: これらの値は本番環境では Secret Manager などのセキュアなストレージで管理し、Git にコミットしないでください。

### 設定例
```bash
# Agent token encryption (for production, use Secret Manager)
AGENT_TOKEN_KEY_V1=your-32-byte-or-longer-secret-key-here-change-this-in-production
AGENT_TOKEN_SALT=your-random-salt-value-here-change-this

# Email credential encryption (for production, use Secret Manager)
EMAIL_TOKEN_KEY_V1=your-32-byte-or-longer-email-key-here-change-this-in-production
EMAIL_TOKEN_SALT=your-random-email-salt-value-here-change-this
# Gmail OAuth credentials (required)
EMAIL_GMAIL_CLIENT_ID=your-client-id
EMAIL_GMAIL_CLIENT_SECRET=your-client-secret
EMAIL_GMAIL_REDIRECT_URL=http://localhost:3000/account_links/callback

# Gmail token file path (optional)
# 未設定の場合は APP_ROOT/credentials/token_user.json が使用されます
# APP_ROOT も未設定の場合は /data/credentials/token_user.json が使用されます
# GMAIL_TOKEN_PATH=/custom/path/to/token_user.json
```
## プロンプトファイルをコピー
プロンプトファイルの内容を変更することで調整できます
```bash
cp /home/dev/backend/prompts/text_analysis_prompt_sample.txt /home/dev/backend/prompts/text_analysis_prompt.txt
```

### 環境変数設定
`.env`ファイルを編集して必要な値を設定：
```bash
OPENAI_API_KEY=生成したOpenAiのAPIキーを記載
LABEL=案件メールが振分済のラベル名を記載
GMAIL_PORT=5555
```
## VsCodeでプロジェクトフォルダーを開く
## Reopen in Containerを押下
もし表示されない場合は Ctrl Shift P→Reopen in containerと入力して実行でもおｋ

## Redis 疎通確認（DevContainer 内）
DevContainer には Redis が含まれています。以下のコマンドで疎通確認ができます：

```bash
redis-cli -h redis -a redis_local_password ping
```

期待される出力:
```
PONG
```

Redis のキーを確認する場合：
```bash
# すべてのキーを表示
redis-cli -h redis -a redis_local_password keys '*'

# レートリミット関連のキーのみ表示
redis-cli -h redis -a redis_local_password keys 'ratelimit:*'
```

## テーブル作成
```bash
task migration-create
```
### 認証URLの生成
Google認証URLを生成します：
```bash
task gmail-auth
```
出力例：
```
Google認証URL:
https://accounts.google.com/o/oauth2/auth?client_id=...&redirect_uri=...

ブラウザでこのURLにアクセスして認証を完了してください。
認証後、リダイレクトURLのcodeパラメータを使用して 'auth-code' コマンドを実行してください。
```

# 環境構築完了です！！
お疲れ様でした。
