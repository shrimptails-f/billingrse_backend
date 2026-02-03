# 概要
マイグレーションの手順と動作確認データ投入手順を解説します。

## 現状の前提
- DB: MySQL
- CI: なし (Atlasはローカル/手動運用)
- 本番・検証環境: 未作成

## マイグレーションの手順
マイグレーションを適用する
```
task migration-create
```
## テストデータ投入手順
テストデータ投入コマンドを実行する
```
task seed
```

## マイグレーション運用について
### テーブル追加時
1. 構造体を作成する
配置場所: /home/dev/backend/tools/migrations/models
構造体にタグを指定することで主キーやユニーク制約を指定できます。
https://gorm.io/ja_JP/docs/models.html#gorm-Model
2. マイグレーション生成コマンドを実行する
```
task migration-gen
```
3. 生成されたDDLを確認し、意図しない破壊的変更がないことを確認する
配置場所: /home/dev/backend/tools/migrations/ddl

### 本番・検証適用
前提: 本番・検証環境でDBが作成されており、各環境で疎通できていること
備考: 現時点では本番・検証が未作成のため、以下は暫定手順
#### 適用手順
1. Dockerfileからイメージをビルド
/home/dev/backend/deploy/db_tools/Dockerfile
2. ECRにプッシュ
3. タスク定義を更新
4. コンテナ内で atlas migrate apply を実行して適用する

#### 運用方針 (暫定)
- CIがないため、適用は手動で実施する
- アプリのローリングデプロイ前にDBマイグレーションを適用する (後方互換前提)
- 破壊的変更が必要な場合は、追加 -> 移行 -> 削除の2段階以上に分ける

#### デプロイ戦略
アプリはローリングデプロイ。DBは先に適用し、後方互換性を保つことを前提とする。

## Atlas関連ファイル
- 設定ファイル: /home/dev/backend/atlas.hcl
- マイグレーションDDL: /home/dev/backend/tools/migrations/ddl
- モデル定義: /home/dev/backend/tools/migrations/models