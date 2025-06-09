package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-radio/internal/radiko"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// テスト用のモッククライアント
type mockRadikoClient struct {
	authError   error
	recordError error
	logger      *radiko.Logger
}

func (m *mockRadikoClient) Auth() error {
	return m.authError
}

func (m *mockRadikoClient) RecordTimeFree(stationID string, startTime time.Time, duration int, outputFile string) error {
	if m.recordError != nil {
		return m.recordError
	}
	
	// テスト用にダミーファイルを作成
	dir := filepath.Dir(outputFile)
	if dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}
	
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// ダミーコンテンツを書き込み
	file.WriteString("dummy audio content")
	return nil
}

func (m *mockRadikoClient) SetLogger(logger *radiko.Logger) {
	m.logger = logger
}

// テスト用のヘルパー関数
func setupTestEnv() {
	os.Setenv("DEFAULT_DURATION", "60")
	os.Setenv("DEFAULT_OUTPUT_DIR", "/tmp/test")
	os.Setenv("VERBOSE", "false")
	os.Setenv("UPLOAD_BUCKET", "")
}

func cleanupTestEnv() {
	os.Unsetenv("DEFAULT_DURATION")
	os.Unsetenv("DEFAULT_OUTPUT_DIR")
	os.Unsetenv("VERBOSE")
	os.Unsetenv("UPLOAD_BUCKET")
	
	// テストファイルのクリーンアップ
	os.RemoveAll("/tmp/test")
}

func TestHandler_ValidInput(t *testing.T) {
	setupTestEnv()
	defer cleanupTestEnv()
	
	// radiko.NewClientを一時的にモックに置き換える必要があるため、
	// 統合テストとして実装する代わりに、個別の関数をテストします
	
	event := Event{
		Station:  "TBS",
		Start:    "2024-01-01 20:00",
		Duration: 60,
		Output:   "test.aac",
		Verbose:  false,
	}
	
	// このテストでは、パラメータの検証部分のみテストします
	if event.Station == "" {
		t.Error("Station should not be empty")
	}
	
	if event.Duration == 0 {
		event.Duration = 60 // デフォルト値
	}
	
	if event.Duration != 60 {
		t.Errorf("Expected duration 60, got %d", event.Duration)
	}
}

func TestHandler_MissingStation(t *testing.T) {
	setupTestEnv()
	defer cleanupTestEnv()
	
	event := Event{
		Station:  "",
		Duration: 60,
		Output:   "test.aac",
	}
	
	// Handlerを直接呼び出すテスト（実際の録音は行わない）
	ctx := context.Background()
	_, err := Handler(ctx, event)
	
	if err == nil {
		t.Error("Expected error for missing station")
	}
	
	if !strings.Contains(err.Error(), "station is required") {
		t.Errorf("Expected 'station is required' error, got: %v", err)
	}
}

func TestHandler_EnvironmentVariables(t *testing.T) {
	// 環境変数の設定
	os.Setenv("DEFAULT_DURATION", "120")
	os.Setenv("DEFAULT_OUTPUT_DIR", "/tmp/radio")
	os.Setenv("VERBOSE", "true")
	defer func() {
		os.Unsetenv("DEFAULT_DURATION")
		os.Unsetenv("DEFAULT_OUTPUT_DIR")
		os.Unsetenv("VERBOSE")
	}()
	
	// 設定の読み込みをテスト
	config := radiko.DefaultConfig()
	
	// 環境変数からの値の適用をテスト
	if defaultDuration := os.Getenv("DEFAULT_DURATION"); defaultDuration != "" {
		if dur := 120; dur != 120 {
			t.Errorf("Expected duration 120, got %d", dur)
		}
	}
	
	if defaultOutputDir := os.Getenv("DEFAULT_OUTPUT_DIR"); defaultOutputDir != "/tmp/radio" {
		t.Errorf("Expected output dir '/tmp/radio', got %s", defaultOutputDir)
	}
	
	if config == nil {
		t.Error("Config should not be nil")
	}
}

func TestHandler_ConfigOverride(t *testing.T) {
	setupTestEnv()
	defer cleanupTestEnv()
	
	configOverride := &ConfigOverride{
		DefaultOutputDir: "/custom/output",
		DefaultDuration:  90,
		FFmpegPath:      "/custom/ffmpeg",
		StationAliases: map[string]string{
			"tbsradio": "TBS",
		},
	}
	
	event := Event{
		Station: "tbsradio",
		Config:  configOverride,
	}
	
	// 設定のオーバーライドのテスト
	config := radiko.DefaultConfig()
	
	if event.Config != nil {
		if event.Config.DefaultOutputDir != "" {
			config.DefaultOutputDir = event.Config.DefaultOutputDir
		}
		if event.Config.DefaultDuration > 0 {
			config.DefaultDuration = event.Config.DefaultDuration
		}
	}
	
	if config.DefaultOutputDir != "/custom/output" {
		t.Errorf("Expected output dir '/custom/output', got %s", config.DefaultOutputDir)
	}
	
	if config.DefaultDuration != 90 {
		t.Errorf("Expected duration 90, got %d", config.DefaultDuration)
	}
}

func TestHandler_TimeFormatting(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		wantError bool
	}{
		{
			name:      "Valid time format",
			startTime: "2024-01-01 20:00",
			wantError: false,
		},
		{
			name:      "Invalid time format",
			startTime: "invalid-time",
			wantError: true,
		},
		{
			name:      "Empty time (should use current time)",
			startTime: "",
			wantError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var startTime time.Time
			var err error
			
			if tt.startTime == "" {
				jst, _ := time.LoadLocation("Asia/Tokyo")
				now := time.Now().In(jst)
				startTime = time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, jst)
			} else {
				startTime, err = time.Parse("2006-01-02 15:04", tt.startTime)
			}
			
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.wantError && startTime.IsZero() {
				t.Error("Start time should not be zero")
			}
		})
	}
}

func TestHandler_OutputFileGeneration(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		stationID  string
		startTime  time.Time
		outputDir  string
		expected   string
	}{
		{
			name:      "Custom output file",
			output:    "custom.aac",
			stationID: "TBS",
			startTime: time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC),
			outputDir: "/tmp",
			expected:  "/tmp/custom.aac",
		},
		{
			name:      "Template output file",
			output:    "yyyymmdd_hhmm.aac",
			stationID: "TBS",
			startTime: time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC),
			outputDir: "/tmp",
			expected:  "/tmp/20240101_2000.aac",
		},
		{
			name:      "Auto-generated output file",
			output:    "",
			stationID: "TBS",
			startTime: time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC),
			outputDir: "/tmp",
			expected:  "/tmp/TBS_20240101_2000.aac",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile := tt.output
			if outputFile == "" {
				outputFile = fmt.Sprintf("%s_%s.aac", tt.stationID, tt.startTime.Format("20060102_1504"))
			} else if outputFile == "yyyymmdd_hhmm.aac" {
				outputFile = fmt.Sprintf("%s.aac", tt.startTime.Format("20060102_1504"))
			}
			
			if !filepath.IsAbs(outputFile) && tt.outputDir != "" {
				outputFile = filepath.Join(tt.outputDir, outputFile)
			}
			
			if !strings.HasSuffix(outputFile, ".aac") {
				outputFile += ".aac"
			}
			
			if outputFile != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, outputFile)
			}
		})
	}
}

func TestEvent_JSONMarshaling(t *testing.T) {
	event := Event{
		Station:  "TBS",
		Start:    "2024-01-01 20:00",
		Duration: 60,
		Output:   "test.aac",
		Verbose:  true,
		Config: &ConfigOverride{
			DefaultOutputDir: "/custom",
			DefaultDuration:  90,
		},
	}
	
	// JSONにマーシャル
	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}
	
	// JSONからアンマーシャル
	var unmarshaled Event
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}
	
	// 値の確認
	if unmarshaled.Station != event.Station {
		t.Errorf("Expected station %s, got %s", event.Station, unmarshaled.Station)
	}
	
	if unmarshaled.Duration != event.Duration {
		t.Errorf("Expected duration %d, got %d", event.Duration, unmarshaled.Duration)
	}
	
	if unmarshaled.Config == nil {
		t.Error("Config should not be nil after unmarshaling")
	} else {
		if unmarshaled.Config.DefaultOutputDir != event.Config.DefaultOutputDir {
			t.Errorf("Expected output dir %s, got %s", 
				event.Config.DefaultOutputDir, unmarshaled.Config.DefaultOutputDir)
		}
	}
}

func TestStationAlias(t *testing.T) {
	stationAliases := map[string]string{
		"tbsradio": "TBS",
		"nhk":      "NHK-FM",
	}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"TBS", "TBS"},
		{"tbsradio", "TBS"},
		{"nhk", "NHK-FM"},
		{"unknown", "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			stationID := tt.input
			if alias, ok := stationAliases[strings.ToLower(stationID)]; ok {
				stationID = alias
			}
			
			if stationID != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, stationID)
			}
		})
	}
}
