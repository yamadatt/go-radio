package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"go-radio/internal/radiko"
)

func main() {
	var (
		stationID  = flag.String("station", "", "ラジオ局ID (例: TBS, LFR)")
		startTime  = flag.String("start", "", "開始時間 (YYYY-MM-DD HH:MM 形式)")
		duration   = flag.Int("duration", 0, "録音時間（分）")
		output     = flag.String("output", "", "出力ファイル名 (.mp3拡張子)")
		listFlag   = flag.Bool("list", false, "利用可能な局の一覧を表示")
		configFlag = flag.Bool("config", false, "設定ファイルを生成")
		verbose    = flag.Bool("verbose", false, "詳細なログを表示")
	)
	flag.Parse()

	// ロガーを初期化
	logger := radiko.NewLogger(*verbose)

	// 設定を読み込み（環境変数も反映）
	config, err := radiko.LoadConfigWithEnv()
	if err != nil {
		logger.Error("設定読み込み警告: %v", err)
	}
	logger.Debug("設定を読み込みました: %+v", config)

	if *configFlag {
		if err := config.SaveConfig(""); err != nil {
			logger.Fatal("設定ファイル生成に失敗: %v", err)
		}
		logger.Info("設定ファイルを生成しました: ~/.go-radio/config.json")
		return
	}

	if *listFlag {
		stations := radiko.GetAvailableStations()
		logger.Info("利用可能なラジオ局:")
		for id, name := range stations {
			logger.Info("  %s: %s", id, name)
		}
		return
	}

	if *stationID == "" || *startTime == "" {
		logger.Info("使用方法:")
		logger.Info("  go run main.go -station=TBS -start=\"2024-06-07 20:00\" -duration=60 -output=program.mp3")
		logger.Info("  go run main.go -list  # 利用可能な局の一覧")
		logger.Info("  go run main.go -config  # 設定ファイル生成")
		logger.Info("  go run main.go -verbose  # 詳細ログを表示")
		os.Exit(1)
	}

	// 局IDのエイリアス処理
	originalStationID := *stationID
	if alias, ok := config.StationAliases[strings.ToLower(*stationID)]; ok {
		*stationID = alias
		logger.Debug("局IDエイリアス: %s -> %s", originalStationID, *stationID)
	}

	// デフォルト録音時間の設定
	if *duration == 0 {
		*duration = config.DefaultDuration
		logger.Debug("デフォルト録音時間を使用: %d分", *duration)
	}

	// 開始時間をパース（常に日本時間として解釈）
	jst, tzErr := time.LoadLocation("Asia/Tokyo")
	if tzErr != nil {
		jst = time.FixedZone("JST", 9*60*60)
	}
	startDateTime, err := time.ParseInLocation("2006-01-02 15:04", *startTime, jst)
	if err != nil {
		logger.Fatal("時間の形式が正しくありません: %v", err)
	}

	// 時間の妥当性をチェック
	if err := radiko.ValidateDateTime(startDateTime); err != nil {
		logger.Fatal("時間の妥当性チェックエラー: %v", err)
	}

	// 出力ファイルパスを決定
	outputFile, err := radiko.BuildOutputPath(config, *stationID, *output, startDateTime)
	if err != nil {
		logger.Fatal("出力パス生成に失敗: %v", err)
	}

	logger.Info("録音設定:")
	logger.Info("  局: %s", *stationID)
	logger.Info("  開始時間: %s", startDateTime.Format("2006-01-02 15:04"))
	logger.Info("  録音時間: %s", radiko.FormatDuration(*duration))
	logger.Info("  出力ファイル: %s", outputFile)

	// Radikoクライアントを作成
	client := radiko.NewClient()
	client.SetLogger(logger)

	// 認証
	logger.Info("radikoクライアント初期化...")
	if err := client.Auth(); err != nil {
		logger.Fatal("クライアント初期化に失敗: %v", err)
	}
	logger.Info("初期化完了")

	// ライブストリーム録音
	logger.Info("ライブストリーム録音を開始...")
	recFile := outputFile
	if strings.HasSuffix(outputFile, ".mp3") {
		recFile = strings.TrimSuffix(outputFile, ".mp3") + ".aac"
	}
	if err := client.RecordTimeFree(*stationID, startDateTime, *duration, recFile); err != nil {
		logger.Fatal("録音に失敗: %v", err)
	}
	if recFile != outputFile {
		if err := radiko.ConvertToMP3(config.FFmpegPath, recFile, outputFile); err != nil {
			logger.Fatal("変換に失敗: %v", err)
		}
		os.Remove(recFile)
	}

	logger.Info("録音完了: %s", outputFile)
}
