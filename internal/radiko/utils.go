package radiko

import (
	"fmt"
	"log"
	"time"
)

// Logger はログ出力を管理
type Logger struct {
	verbose bool
}

// NewLogger は新しいLoggerを作成
func NewLogger(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

// Info は情報ログを出力
func (l *Logger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Debug はデバッグログを出力（verboseモードでのみ）
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Error はエラーログを出力
func (l *Logger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Fatal はエラーログを出力して終了
func (l *Logger) Fatal(format string, args ...interface{}) {
	log.Fatalf("[FATAL] "+format, args...)
}

// ValidateDateTime は日時の妥当性をチェック
func ValidateDateTime(startTime time.Time) error {
	now := time.Now()

	// 過去1週間以内かチェック
	weekAgo := now.AddDate(0, 0, -7)
	if startTime.Before(weekAgo) {
		return fmt.Errorf("開始時間が古すぎます。タイムフリーは過去1週間分のみ利用可能です")
	}

	// 未来の時間でないかチェック
	if startTime.After(now) {
		return fmt.Errorf("未来の時間は指定できません")
	}

	return nil
}

// FormatDuration は時間を読みやすい形式に変換
func FormatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d分", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%d時間", hours)
	}
	return fmt.Sprintf("%d時間%d分", hours, mins)
}
