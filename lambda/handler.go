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
)

// Event defines input parameters for the Lambda function
type Event struct {
    Station  string `json:"station"`
    Start    string `json:"start"`
    Duration int    `json:"duration"`
    Output   string `json:"output"`
    Verbose  bool   `json:"verbose"`
}

// Handler is the Lambda entry point
func Handler(ctx context.Context, e Event) (string, error) {
    logger := radiko.NewLogger(e.Verbose)

    config, err := radiko.LoadConfig("")
    if err != nil {
        logger.Error("設定読み込み警告: %v", err)
        config = radiko.DefaultConfig()
    }

    if e.Station == "" || e.Start == "" {
        return "", fmt.Errorf("station and start are required")
    }

    stationID := e.Station
    if alias, ok := config.StationAliases[strings.ToLower(stationID)]; ok {
        stationID = alias
    }

    duration := e.Duration
    if duration == 0 {
        duration = config.DefaultDuration
    }

    startTime, err := time.Parse("2006-01-02 15:04", e.Start)
    if err != nil {
        return "", fmt.Errorf("時間の形式が正しくありません: %w", err)
    }

    if err := radiko.ValidateDateTime(startTime); err != nil {
        return "", err
    }

    outputFile := e.Output
    if outputFile == "" {
        outputFile = fmt.Sprintf("%s_%s.aac", stationID, startTime.Format("20060102_1504"))
        if config.DefaultOutputDir != "" {
            outputFile = filepath.Join(config.DefaultOutputDir, outputFile)
        }
    }
    if !strings.HasSuffix(outputFile, ".aac") {
        outputFile += ".aac"
    }

    outputDir := filepath.Dir(outputFile)
    if outputDir != "." {
        if err := os.MkdirAll(outputDir, 0755); err != nil {
            return "", fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
        }
    }

    client := radiko.NewClient()
    client.SetLogger(logger)
    if err := client.Auth(); err != nil {
        return "", fmt.Errorf("クライアント初期化に失敗: %w", err)
    }

    if err := client.RecordTimeFree(stationID, startTime, duration, outputFile); err != nil {
        return "", fmt.Errorf("録音に失敗: %w", err)
    }

    return fmt.Sprintf("録音完了: %s", outputFile), nil
}

func main() {
    lambda.Start(Handler)
}

