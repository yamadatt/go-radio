package radiko

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultDuration != 60 {
		t.Errorf("expected default duration 60, got %d", cfg.DefaultDuration)
	}
	home, _ := os.UserHomeDir()
	expectedDir := filepath.Join(home, "Downloads", "radiko")
	if cfg.DefaultOutputDir != expectedDir {
		t.Errorf("expected output dir %s, got %s", expectedDir, cfg.DefaultOutputDir)
	}
	if cfg.FFmpegPath != "ffmpeg" {
		t.Errorf("expected ffmpeg path ffmpeg, got %s", cfg.FFmpegPath)
	}
	if len(cfg.StationAliases) == 0 {
		t.Errorf("station aliases should not be empty")
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	cfg := &Config{
		DefaultOutputDir: "/tmp/output",
		DefaultDuration:  30,
		StationAliases:   map[string]string{"x": "y"},
		FFmpegPath:       "/usr/bin/ffmpeg",
	}
	if err := cfg.SaveConfig(path); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if loaded.DefaultOutputDir != cfg.DefaultOutputDir || loaded.DefaultDuration != cfg.DefaultDuration || loaded.FFmpegPath != cfg.FFmpegPath {
		t.Errorf("basic fields not restored")
	}
	if loaded.StationAliases["x"] != "y" {
		t.Errorf("custom alias missing in loaded config")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	def := DefaultConfig()
	if cfg.DefaultDuration != def.DefaultDuration {
		t.Errorf("expected default duration %d, got %d", def.DefaultDuration, cfg.DefaultDuration)
	}
}
