# Go-Radio - Radikoタイムフリー録音ツール

Golangで作成されたradikoのタイムフリー番組を音声ファイル（AAC形式）として保存するツールです。CLIアプリケーションとしてもAWS Lambda 関数としても利用できます。

オンプレ（自宅）で録音するのではなく、クラウドで録音しようと思って作った。

Lambdaで動かすことで、安価に運用できることを期待している。

## 機能

- radikoのタイムフリー番組を録音
- 指定した時間での録音
- AAC形式での音声ファイル出力
- 利用可能なラジオ局の一覧表示

## 必要な環境

 - Go 1.24以上

## インストール

```bash
git clone <repository-url>
cd go-radio
go mod tidy
```

## 認証フロー

1. `https://radiko.jp/v2/api/auth1` にアクセスし、レスポンスヘッダーから
   `X-Radiko-Authtoken` と `X-Radiko-KeyLength` / `X-Radiko-KeyOffset` を取得します。
2. ラジコ公式の JavaScript に埋め込まれている共通鍵を、取得したオフセットと長さで
   切り出して Base64 エンコードしたものを `X-Radiko-Partialkey` として生成します。
3. `https://radiko.jp/v2/api/auth2` に `X-Radiko-Authtoken` と `X-Radiko-Partialkey`
   をヘッダーに付与してリクエストすると、トークンが有効化され、レスポンス本文に
   エリアIDなどが返ります。
4. 以降はストリームやタイムフリーの M3U8 を取得する際に `X-Radiko-Authtoken` を
   ヘッダーに付けてアクセスします。

## ファイル構成

```
go-radio/
├── README.md              # このドキュメント
├── go.mod                 # Go モジュール定義
├── main.go                # CLIエントリポイント
├── lambda/
│   └── handler.go         # AWS Lambda ハンドラー
├── template.yaml          # SAM テンプレート
├── Dockerfile             # Lambda用 Dockerfile
└── internal/
    └── radiko/
        ├── client.go      # Radiko API クライアント
        ├── config.go      # 設定関連
        └── utils.go       # 補助関数
```

## 設定ファイル

`-config` オプションを実行すると `~/.go-radio/config.json` にデフォルト設定を生成できます。

```bash
go run main.go -config
```

設定ファイルでは出力ディレクトリやデフォルト録音時間、局IDのエイリアスなどを指定できます。

## 使用方法

### 基本的な使用方法

```bash
go run main.go -station=TBS -start="2024-06-07 20:00" -duration=60 -output=program.aac
```

### パラメータ

- `-station`: ラジオ局ID（必須）
- `-start`: 録音開始時間（必須、YYYY-MM-DD HH:MM形式）
- `-duration`: 録音時間（分、デフォルト: 60分）
- `-output`: 出力ファイル名（省略時は自動生成）
- `-list`: 利用可能な局の一覧を表示
- `-config`: デフォルト設定ファイルを生成
- `-verbose`: 詳細ログを表示

### 利用可能なラジオ局の確認

```bash
go run main.go -list
```

### 使用例

#### TBSラジオの番組を1時間録音
```bash
go run main.go -station=TBS -start="2024-06-07 20:00" -duration=60
```

#### ニッポン放送の番組を30分録音（出力ファイル名指定）
```bash
go run main.go -station=LFR -start="2024-06-07 21:00" -duration=30 -output=nippon_program.aac
```

#### J-WAVEの番組を2時間録音
```bash
go run main.go -station=FMJ -start="2024-06-07 18:00" -duration=120
```

## 対応ラジオ局

- TBS: TBSラジオ
- LFR: ニッポン放送
- QRR: 文化放送
- RN1: ラジオNIKKEI第1
- RN2: ラジオNIKKEI第2
- INT: interfm
- FMT: TOKYO FM
- FMJ: J-WAVE
- JORF: ラジオ日本
- BAYFM: bayfm
- NACK5: NACK5
- YFM: FM YOKOHAMA

## 注意事項

- タイムフリー機能を利用して指定した時間の番組を録音できます
- 録音した音声ファイルは個人的な用途のみに使用してください
- 著作権法を遵守してご利用ください
- radikoの利用規約に従ってご利用ください

## ビルド

実行可能ファイルを作成する場合：

```bash
go build -o go-radio main.go
./go-radio -station=TBS -start="2024-06-07 20:00" -duration=60
```

## AWS Lambda デプロイ (SAM + Docker)

このリポジトリには `Dockerfile` と `template.yaml` が含まれており、SAM CLI を用いたコンテナベースの Lambda デプロイが可能です。

1. SAM CLI をインストールします。
2. `sam build` を実行してイメージをビルドします。
3. `sam deploy --guided` を実行し、デプロイ情報を入力します。
   `UPLOAD_BUCKET` 環境変数にアップロード先の S3 バケット名を指定してください。
   例:`UPLOAD_BUCKET=radio-transcribe`

イベントの例:
```json
{
  "station": "TBS",
  "start": "2024-06-07 20:00",
  "duration": 60,
  "output": "program.aac"
}
```

### 環境変数

- `VERBOSE` - `true` を指定すると詳細ログを出力します
- `DEFAULT_DURATION` - 録音時間のデフォルト値を上書きします
- `DEFAULT_OUTPUT_DIR` - 相対パス指定時に付与する出力ディレクトリ
- `UPLOAD_BUCKET` - 録音後にファイルをアップロードする S3 バケット名

`output` に相対パスを指定した場合、Lambda 実行環境では `DEFAULT_OUTPUT_DIR`
（デフォルト `/tmp/radiko`）が自動的に付与されます。書き込みエラーが発生する
場合は `/tmp` 以下のディレクトリを指定してください。


## トラブルシューティング

### 認証エラーが発生する場合
- ネットワーク接続を確認してください
- radikoのサービスが正常に動作しているか確認してください
- Auth1/Auth2 を用いた公式の認証フローを実装しています。X-Radiko-Authtoken
  が無効な場合は、再度 `go run` を実行して認証をやり直してください。

### 番組が見つからない場合
- 指定した時間に番組が放送されていたか確認してください
- タイムフリーの利用可能期間（過去1週間）内かどうか確認してください
- `録音に失敗: open *.aac: read-only file system` というエラーが出る場合は、
  `/tmp` 以下に書き込むよう `output` を設定するか、`DEFAULT_OUTPUT_DIR`
  を `/tmp` 以下に指定してください

## ライセンス

このソフトウェアは個人的な学習目的で作成されています。radikoの利用規約を遵守してご利用ください。

## テスト（参考）

基本的なJSONイベント。マネコンのLambdaテストで以下のパラメータを設定する。

```json
{
  "station": "TBS",
  "start": "2025-06-08 20:00",
  "duration": 60,
  "output": "program.aac"
}
```

`test-lambda.json` として同じ内容のサンプルイベントをリポジトリに含めています。

AWS CLI / SDK での実行。

```bash
aws lambda invoke \
  --function-name RadioFunction \
  --payload '{"station":"TBS","start":"2025-06-08 20:00","duration":60}' \
  response.json
```