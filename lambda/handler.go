package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-radio/internal/radiko"

	"github.com/aws/aws-lambda-go/lambda"

	// S3 upload
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Event defines input parameters for the Lambda function
type Event struct {
	Station  string `json:"station"`
	Start    string `json:"start"`
	Duration int    `json:"duration"`
	Output   string `json:"output"`
	Verbose  bool   `json:"verbose"`

	// 追加の設定オプション
	Config *ConfigOverride `json:"config,omitempty"`
}

// ConfigOverride allows overriding default configuration
type ConfigOverride struct {
	DefaultOutputDir string            `json:"default_output_dir,omitempty"`
	DefaultDuration  int               `json:"default_duration,omitempty"`
	StationAliases   map[string]string `json:"station_aliases,omitempty"`
	FFmpegPath       string            `json:"ffmpeg_path,omitempty"`
}

// Handler is the Lambda entry point
func Handler(ctx context.Context, e Event) (string, error) {
	// 環境変数からverboseを取得（イベントで上書き可能）
	verbose := e.Verbose
	if os.Getenv("VERBOSE") == "true" {
		verbose = true
	}

	logger := radiko.NewLogger(verbose)

	// 設定を読み込み（環境変数も反映）
	config, err := radiko.LoadConfigWithEnv()
	if err != nil {
		logger.Error("設定読み込み警告: %v", err)
	}

	// イベントから設定をオーバーライド
	if e.Config != nil {
		if e.Config.DefaultOutputDir != "" {
			config.DefaultOutputDir = e.Config.DefaultOutputDir
		}
		if e.Config.DefaultDuration > 0 {
			config.DefaultDuration = e.Config.DefaultDuration
		}
		if e.Config.FFmpegPath != "" {
			config.FFmpegPath = e.Config.FFmpegPath
		}
		if e.Config.StationAliases != nil {
			for k, v := range e.Config.StationAliases {
				config.StationAliases[k] = v
			}
		}
	}

	if e.Station == "" {
		return "", fmt.Errorf("station is required")
	}

	stationID := e.Station
	if alias, ok := config.StationAliases[strings.ToLower(stationID)]; ok {
		stationID = alias
	}

	duration := e.Duration
	if duration == 0 {
		duration = config.DefaultDuration
	}

	var startTime time.Time
	jst, tzErr := time.LoadLocation("Asia/Tokyo")
	if tzErr != nil {
		jst = time.FixedZone("JST", 9*60*60)
	}

	if e.Start == "" {
		now := time.Now().In(jst)
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, jst)
	} else {
		var err error
		startTime, err = time.ParseInLocation("2006-01-02 15:04", e.Start, jst)
		if err != nil {
			return "", fmt.Errorf("時間の形式が正しくありません: %w", err)
		}
	}

	if err := radiko.ValidateDateTime(startTime); err != nil {
		return "", err
	}

	// 出力ファイルパスを決定
	outputFile, err := radiko.BuildOutputPath(config, stationID, e.Output, startTime)
	if err != nil {
		return "", fmt.Errorf("出力パス生成に失敗: %w", err)
	}

	client := radiko.NewClient()
	client.SetLogger(logger)
	if err := client.Auth(); err != nil {
		return "", fmt.Errorf("クライアント初期化に失敗: %w", err)
	}

	if err := client.RecordTimeFree(stationID, startTime, duration, outputFile); err != nil {
		return "", fmt.Errorf("録音に失敗: %w", err)
	}

	// Upload to S3 if bucket is specified
	bucket := os.Getenv("UPLOAD_BUCKET")
	if bucket != "" {
		key := filepath.Base(outputFile)
		if err := uploadFileToS3(ctx, bucket, key, outputFile); err != nil {
			return "", fmt.Errorf("S3アップロード失敗: %w", err)
		}
	}

	return fmt.Sprintf("録音完了: %s", outputFile), nil
}

func uploadFileToS3(ctx context.Context, bucket, key, path string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(cfg)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String("audio/aac"),
	})
	return err
}

func main() {
	lambda.Start(Handler)
}
