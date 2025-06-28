package radiko

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LoadConfigWithEnv loads the configuration file and applies
// overrides from environment variables. It falls back to the
// default configuration if loading fails.
func LoadConfigWithEnv() (*Config, error) {
	cfg, err := LoadConfig("")
	if err != nil {
		cfg = DefaultConfig()
	}

	if v := os.Getenv("DEFAULT_DURATION"); v != "" {
		if d, convErr := strconv.Atoi(v); convErr == nil {
			cfg.DefaultDuration = d
		}
	}

	if v := os.Getenv("DEFAULT_OUTPUT_DIR"); v != "" {
		cfg.DefaultOutputDir = v
	}

	if v := os.Getenv("FFMPEG_PATH"); v != "" {
		cfg.FFmpegPath = v
	}

	return cfg, err
}

// BuildOutputPath creates the final output file path based on the
// provided parameters and configuration. The directory part of the
// path is created if necessary.
func BuildOutputPath(cfg *Config, stationID, output string, start time.Time) (string, error) {
	file := output
	if file == "" {
		file = fmt.Sprintf("%s_%s.mp3", stationID, start.Format("20060102_1504"))
	} else if file == "yyyymmdd_hhmm.mp3" {
		file = fmt.Sprintf("%s.mp3", start.Format("20060102_1504"))
	}

	if !filepath.IsAbs(file) && cfg.DefaultOutputDir != "" {
		file = filepath.Join(cfg.DefaultOutputDir, file)
	}

	if !strings.HasSuffix(file, ".mp3") {
		file += ".mp3"
	}

	dir := filepath.Dir(file)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	return file, nil
}
