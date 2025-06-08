package radiko

import (
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
