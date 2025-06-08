package radiko

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateDateTime(t *testing.T) {
	now := time.Now()
	if err := ValidateDateTime(now.Add(-time.Hour)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateDateTime(now.AddDate(0, 0, -8)); err == nil {
		t.Errorf("expected error for old time")
	}
	if err := ValidateDateTime(now.Add(time.Hour)); err == nil {
		t.Errorf("expected error for future time")
	}
}

func TestCheckFFmpeg(t *testing.T) {
	dir := t.TempDir()
	ff := filepath.Join(dir, "ffmpeg")
	if err := os.WriteFile(ff, []byte(""), 0755); err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := CheckFFmpeg(ff); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := CheckFFmpeg(filepath.Join(dir, "missing")); err == nil {
		t.Errorf("expected error for missing file")
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(30); got != "30分" {
		t.Errorf("unexpected result: %s", got)
	}
	if got := FormatDuration(60); got != "1時間" {
		t.Errorf("unexpected result: %s", got)
	}
	if got := FormatDuration(135); got != "2時間15分" {
		t.Errorf("unexpected result: %s", got)
	}
}
