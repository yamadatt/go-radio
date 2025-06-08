# Go-Radio - Radikoタイムフリー録音ツール

Golangで作成されたradikoのタイムフリー番組を音声ファイル（AAC形式）として保存するツールです。

## 機能

- radikoのライブストリーム録音（タイムフリーの代替）
- 指定した時間での録音
- AAC形式での音声ファイル出力
- 利用可能なラジオ局の一覧表示

## 必要な環境

- Go 1.19以上
- ffmpeg（音声ファイルのダウンロードに使用）

### ffmpegのインストール

macOSの場合：
```bash
brew install ffmpeg
```

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
├── main.go                # エントリポイント
└── internal/
    └── radiko/
        ├── client.go      # Radiko API クライアント
        ├── config.go      # 設定関連
        └── utils.go       # 補助関数
```

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

- 現在はライブストリームの録音のみ対応しています（タイムフリー機能は制限があります）
- 指定した開始時間に関係なく、現在放送中の番組を録音します
- 録音した音声ファイルは個人的な用途のみに使用してください
- 著作権法を遵守してご利用ください
- radikoの利用規約に従ってご利用ください

## ビルド

実行可能ファイルを作成する場合：

```bash
go build -o go-radio main.go
./go-radio -station=TBS -start="2024-06-07 20:00" -duration=60
```

## トラブルシューティング

### ffmpegが見つからない場合
```
ffmpegがインストールされていません。brew install ffmpegでインストールしてください
```
→ ffmpegをインストールしてください

### 認証エラーが発生する場合
- ネットワーク接続を確認してください
- radikoのサービスが正常に動作しているか確認してください
- Auth1/Auth2 を用いた公式の認証フローを実装しています。X-Radiko-Authtoken
  が無効な場合は、再度 `go run` を実行して認証をやり直してください。

### 番組が見つからない場合
- 指定した時間に番組が放送されていたか確認してください
- タイムフリーの利用可能期間（過去1週間）内かどうか確認してください

## ライセンス

このソフトウェアは個人的な学習目的で作成されています。radikoの利用規約を遵守してご利用ください。
