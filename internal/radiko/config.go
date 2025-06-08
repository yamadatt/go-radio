package radiko

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config はアプリケーションの設定
type Config struct {
	DefaultOutputDir string            `json:"default_output_dir"`
	DefaultDuration  int               `json:"default_duration"`
	StationAliases   map[string]string `json:"station_aliases"`
	FFmpegPath       string            `json:"ffmpeg_path"`
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		DefaultOutputDir: filepath.Join(homeDir, "Downloads", "radiko"),
		DefaultDuration:  60,
		StationAliases: map[string]string{
			"tbs":     "TBS",
			"nippon":  "LFR",
			"bunka":   "QRR",
			"nikkei1": "RN1",
			"nikkei2": "RN2",
			"inter":   "INT",
			"tfm":     "FMT",
			"jwave":   "FMJ",
			"radionippon": "JORF",
			"bay":     "BAYFM",
			"nack":    "NACK5",
			"fmy":     "YFM",
		},
		FFmpegPath: "ffmpeg",
	}
}

// LoadConfig は設定ファイルを読み込む
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()
	
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return config, nil
		}
		configPath = filepath.Join(homeDir, ".go-radio", "config.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, config)
	return config, err
}

// SaveConfig は設定ファイルを保存
func (c *Config) SaveConfig(configPath string) error {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir := filepath.Join(homeDir, ".go-radio")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
		configPath = filepath.Join(configDir, "config.json")
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
