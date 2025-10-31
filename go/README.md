# Gmail-LINE 通知アプリ

Gmail APIとLINE Bot APIを連携させて、新着メールをLINEに通知するアプリケーションです。

## 機能

- Gmail APIを使用してメールを取得
- 新着メールが届いたらLINEに通知を送信
- Gmail Pub/Sub通知をWebhookで受信（リアルタイム通知）

## アーキテクチャ

```
go/
├── cmd/server/          # アプリケーションのエントリーポイント
├── internal/
│   ├── domain/          # ドメインモデルとインターフェース
│   │   ├── gmail/       # Gmail関連のドメインモデル
│   │   └── line/        # LINE関連のドメインモデル
│   ├── infrastructure/  # 外部サービスとの連携
│   │   └── repository/
│   │       ├── gmail/   # Gmail API実装
│   │       └── line/    # LINE Bot API実装
│   ├── service/         # ビジネスロジック
│   │   └── notification/ # 通知サービス
│   └── handler/         # HTTPハンドラー
│       └── webhook/     # Gmail Webhook処理
```

## セットアップ手順

### 1. 前提条件

- Go 1.24.4以降
- Gmail アカウント
- LINE Developerアカウント

### 2. Google Cloud Platformの設定

#### Gmail APIの有効化

1. [Google Cloud Console](https://console.cloud.google.com/)にアクセス
2. 新しいプロジェクトを作成するか、既存のプロジェクトを選択
3. 「APIとサービス」→「ライブラリ」から「Gmail API」を検索して有効化
4. 「認証情報」→「認証情報を作成」→「OAuth 2.0 クライアント ID」を選択
5. アプリケーションの種類で「デスクトップアプリ」を選択
6. 作成した認証情報をJSON形式でダウンロードし、`go/credentials.json`として保存

#### Cloud Pub/Sub の設定（オプション：リアルタイム通知用）

1. Google Cloud Consoleで「Pub/Sub」→「トピック」を選択
2. 新しいトピックを作成（例：`gmail-notifications`）
3. Gmail APIにトピックへの公開権限を付与：
   ```bash
   gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
     --member=serviceAccount:gmail-api-push@system.gserviceaccount.com \
     --role=roles/pubsub.publisher
   ```
4. トピックのサブスクリプションを作成（Pushサブスクリプション）
    - エンドポイントURL: `https://your-domain.com/webhook/gmail`

### 3. LINE Developerの設定

#### LINE Bot の作成

1. [LINE Developers Console](https://developers.line.biz/console/)にアクセス
2. 新規プロバイダーを作成（既存のものを使用してもOK）
3. 「Messaging API」チャネルを作成
4. 「Channel access token」を発行してメモ
5. LINE公式アカウントを友だち追加
6. ユーザーIDを取得：
    - Webhookを一時的に設定し、テストメッセージを送信
    - または、LINE Bot SDKのツールを使用してユーザーIDを確認

### 4. 環境変数の設定

`.env.example`をコピーして`.env`ファイルを作成：

```bash
cd go
cp .env.example .env
```

`.env`ファイルを編集して、以下の値を設定：

```bash
# サーバーポート
PORT=8080

# LINE Bot設定
LINE_CHANNEL_TOKEN=your_line_channel_token_here
LINE_USER_ID=your_line_user_id_here

# Gmail API設定
GMAIL_CREDENTIALS_PATH=./credentials.json
GMAIL_TOKEN_PATH=./token.json

# Pub/Sub設定（オプション）
PUBSUB_TOPIC_NAME=projects/your-project-id/topics/gmail-notifications
```

### 5. 依存関係のインストール

```bash
cd go
go mod download
```

### 6. アプリケーションの起動

```bash
# 初回起動時はOAuth認証が必要
go run cmd/server/main.go
```

初回起動時には、ブラウザでGoogleアカウントの認証が求められます。
認証を完了すると、`token.json`が生成され、次回以降は自動的に認証されます。

### 7. Webhookの設定（本番環境）

本番環境でリアルタイム通知を受け取るには：

1. アプリケーションを公開URLでホスティング（例：Cloud Run、Heroku等）
2. Google Cloud Pub/SubのPushサブスクリプションのエンドポイントを設定：
   ```
   https://your-domain.com/webhook/gmail
   ```
3. Gmail APIでメールボックスの監視を開始（アプリ起動時に自動実行）

## 使用方法

### メール通知のテスト

1. アプリケーションを起動
2. 設定したGmailアカウントにメールを送信
3. Gmail Pub/Sub通知がWebhookエンドポイントに届く
4. LINEに通知が届くことを確認

### エンドポイント

- `GET /health` - ヘルスチェック
- `POST /webhook/gmail` - Gmail Pub/Sub Webhook受信

## トラブルシューティング

### Gmail APIの認証エラー

- `credentials.json`が正しい場所にあることを確認
- Google Cloud Consoleで「Gmail API」が有効化されていることを確認
- OAuth同意画面の設定を確認

### LINE通知が届かない

- `LINE_CHANNEL_TOKEN`が正しいことを確認
- `LINE_USER_ID`が正しいことを確認
- LINE公式アカウントをブロックしていないことを確認

### Pub/Sub通知が届かない

- Pub/Subのトピックとサブスクリプションが正しく設定されていることを確認
- サブスクリプションのエンドポイントURLが正しいことを確認
- Gmail API push権限が付与されていることを確認

## 開発

### ビルド

```bash
cd go
go build -o bin/gmail-line-bot cmd/server/main.go
```

### テスト実行

```bash
cd go
go test ./...
```

## ライセンス

MIT

## 参考資料

- [Gmail API Documentation](https://developers.google.com/gmail/api)
- [LINE Messaging API Documentation](https://developers.line.biz/ja/docs/messaging-api/)
- [Google Cloud Pub/Sub Documentation](https://cloud.google.com/pubsub/docs)
