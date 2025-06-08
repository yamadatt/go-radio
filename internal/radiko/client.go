package radiko

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Client はradikoクライアント
type Client struct {
	authToken  string
	httpClient *http.Client
	logger     *Logger
}

// NewClient は新しいradikoクライアントを作成
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: NewLogger(false), // デフォルトはverbose=false
	}
}

// SetLogger はロガーを設定
func (c *Client) SetLogger(logger *Logger) {
	c.logger = logger
}

// GetAvailableStations は利用可能な局の一覧を返す
func GetAvailableStations() map[string]string {
	return map[string]string{
		"TBS":  "TBSラジオ",
		"LFR":  "ニッポン放送",
		"QRR":  "文化放送",
		"RN1":  "ラジオNIKKEI第1",
		"RN2":  "ラジオNIKKEI第2",
		"INT":  "interfm",
		"FMT":  "TOKYO FM",
		"FMJ":  "J-WAVE",
		"JORF": "ラジオ日本",
		"BAYFM": "bayfm",
		"NACK5": "NACK5",
		"YFM":   "FM YOKOHAMA",
	}
}

// Auth はradikoの認証を行う
func (c *Client) Auth() error {
	c.logger.Debug("radiko認証を開始")
	
	// Step 1: auth1 - 認証トークンを取得
	auth1URL := "https://radiko.jp/v2/api/auth1_fms"
	req, err := http.NewRequest("POST", auth1URL, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("auth1リクエスト作成エラー: %w", err)
	}
	
	// 必要なヘッダーを設定
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Radiko-App", "pc_ts")
	req.Header.Set("X-Radiko-App-Version", "4.0.0")
	req.Header.Set("X-Radiko-User", "test-stream")
	req.Header.Set("X-Radiko-Device", "pc")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	
	c.logger.Debug("auth1リクエスト送信中...")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth1リクエストエラー: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("auth1レスポンス: HTTP %d", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		// レスポンスボディを読み取ってデバッグ情報として表示
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		c.logger.Debug("auth1エラーレスポンス: %s", buf.String())
		return fmt.Errorf("auth1認証失敗: HTTP %d", resp.StatusCode)
	}
	
	// 認証トークンを取得
	authToken := resp.Header.Get("x-radiko-authtoken")
	if authToken == "" {
		authToken = resp.Header.Get("X-Radiko-AuthToken")
	}
	if authToken == "" {
		return fmt.Errorf("認証トークンが取得できませんでした")
	}
	
	c.authToken = authToken
	// トークンの表示用に安全な長さを計算
	tokenDisplayLength := len(authToken)
	if tokenDisplayLength > 10 {
		tokenDisplayLength = 10
	}
	c.logger.Debug("認証トークン取得完了: %s", authToken[:tokenDisplayLength]+"...")
	
	// レスポンスボディから追加情報を取得
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	authResponse := buf.String()
	c.logger.Debug("auth1レスポンス内容: %s", authResponse)
	
	c.logger.Info("radiko認証完了")
	return nil
}

// generatePartialKey は不要（認証スキップ）
// このメソッドは削除されました

// RecordTimeFree はライブストリーム番組を録音（タイムフリーの代替）
func (c *Client) RecordTimeFree(stationID string, startTime time.Time, duration int, outputFile string) error {
	c.logger.Info("注意: タイムフリーではなくライブストリームを録音します")
	
	c.logger.Debug("録音設定: 局=%s, 録音時間=%d分", stationID, duration)
	
	// ライブストリーミングURLを取得
	streamURL, err := c.getTimeFreeURL(stationID, startTime, time.Time{})
	if err != nil {
		return fmt.Errorf("ストリーミングURL取得エラー: %w", err)
	}

	c.logger.Info("ストリーミングURL取得完了")
	c.logger.Debug("ストリーミングURL: %s", streamURL)
	c.logger.Info("ダウンロード開始...")

	// ffmpegを使用して音声ファイルをダウンロード
	return c.downloadWithFFmpeg(streamURL, outputFile, duration)
}

// getTimeFreeURL はライブストリーミングURLを取得
func (c *Client) getTimeFreeURL(stationID string, startTime, endTime time.Time) (string, error) {
	c.logger.Debug("ストリーミングURL取得を開始: 局=%s", stationID)
	
	// 認証が必要
	if c.authToken == "" {
		return "", fmt.Errorf("認証が必要です。先にAuth()を実行してください")
	}
	
	// ライブストリーミング用のプレイリストURL取得
	// radikoの実際のライブストリームエンドポイント
	streamInfoURL := fmt.Sprintf("https://radiko.jp/v2/api/ts/playlist.m3u8?station_id=%s&l=15&lsid=%d&type=b", 
		stationID, time.Now().Unix())
	
	req, err := http.NewRequest("GET", streamInfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("ストリーミング情報リクエスト作成エラー: %w", err)
	}
	
	// 認証ヘッダーを設定
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Radiko-AuthToken", c.authToken)
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	
	c.logger.Debug("ストリーミング情報リクエスト送信中: %s", streamInfoURL)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ストリーミング情報リクエストエラー: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("ストリーミングレスポンス: HTTP %d", resp.StatusCode)
	
	if resp.StatusCode != 200 {
		// エラーレスポンスの内容を表示
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		c.logger.Debug("ストリーミングエラーレスポンス: %s", buf.String())
		return "", fmt.Errorf("ストリーミング情報取得失敗: HTTP %d", resp.StatusCode)
	}
	
	// レスポンスを読み取り（m3u8プレイリスト）
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ストリーミング情報読み取りエラー: %w", err)
	}
	
	playlistContent := buf.String()
	c.logger.Debug("プレイリスト取得完了、内容長: %d bytes", len(playlistContent))
	
	// プレイリスト内容の表示用に安全な長さを計算
	displayLength := len(playlistContent)
	if displayLength > 200 {
		displayLength = 200
	}
	c.logger.Debug("プレイリスト内容: %s", playlistContent[:displayLength]+"...")
	
	// プレイリストから実際のストリーミングURLを抽出
	lines := strings.Split(playlistContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "https://") && strings.Contains(line, ".m3u8") {
			c.logger.Debug("ストリーミングURL見つかりました: %s", line)
			return line, nil
		}
	}
	
	// プレイリストに直接URLが含まれていない場合、元のURLを返す
	c.logger.Debug("直接プレイリストURLを使用: %s", streamInfoURL)
	return streamInfoURL, nil
}

// downloadWithFFmpeg はffmpegを使用して音声をダウンロード
func (c *Client) downloadWithFFmpeg(streamURL, outputFile string, duration int) error {
	// ffmpegがインストールされているかチェック
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpegがインストールされていません。brew install ffmpegでインストールしてください")
	}

	// 出力ディレクトリが存在するか確認
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("出力ディレクトリ作成エラー: %w", err)
		}
	}

	// ffmpegコマンドを実行（HLS/m3u8対応）
	args := []string{
		"-i", streamURL,
		"-t", fmt.Sprintf("%d", duration*60), // 秒に変換
		"-c:a", "aac", // AACコーデックを明示的に指定
		"-b:a", "128k", // ビットレート設定
		"-f", "adts", // AAC形式
		"-y", // 既存ファイルを上書き
		"-loglevel", "warning", // ログレベルを制限
		outputFile,
	}

	// radikoのストリーミング用の追加ヘッダー
	if strings.Contains(streamURL, "radiko.jp") {
		headerArgs := []string{
			"-headers", "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		}
		// ヘッダーを最初に追加
		args = append(headerArgs, args...)
	}

	c.logger.Debug("ffmpegコマンド: ffmpeg %s", strings.Join(args, " "))
	cmd := exec.Command("ffmpeg", args...)
	
	// プログレス表示のため、stderrを取得
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	c.logger.Info("ffmpegでダウンロード中（%d分間）...", duration)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg実行エラー: %w\nstderr: %s", err, stderr.String())
	}

	// ファイルサイズをチェック
	fileInfo, err := os.Stat(outputFile)
	if err != nil {
		return fmt.Errorf("出力ファイル確認エラー: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("ダウンロードされたファイルが空です")
	}

	c.logger.Info("ダウンロード完了: %s (%.2f MB)", outputFile, float64(fileInfo.Size())/(1024*1024))
	return nil
}
