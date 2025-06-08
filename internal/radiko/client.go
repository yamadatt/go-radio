package radiko

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// Auth はradikoの認証を行う（正式な認証フロー）
func (c *Client) Auth() error {
	c.logger.Debug("radiko認証を開始")
	fmt.Printf("=== RADIKO認証デバッグ開始 ===\n")
	
	// Step 1: auth1 - 認証トークンとキー情報を取得
	auth1URL := "https://radiko.jp/v2/api/auth1"
	fmt.Printf("auth1 URL: %s\n", auth1URL)
	
	req, err := http.NewRequest("POST", auth1URL, nil)
	if err != nil {
		return fmt.Errorf("auth1リクエスト作成エラー: %w", err)
	}
	
	// 必要なヘッダーを設定
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Radiko-App", "pc_ts")
	req.Header.Set("X-Radiko-App-Version", "4.0.0")
	req.Header.Set("X-Radiko-User", "test-stream")
	req.Header.Set("X-Radiko-Device", "pc")
	
	fmt.Printf("auth1リクエスト送信中...\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("auth1リクエストエラー: %v\n", err)
		return fmt.Errorf("auth1リクエストエラー: %w", err)
	}
	defer resp.Body.Close()
	
	fmt.Printf("auth1レスポンス: HTTP %d\n", resp.StatusCode)
	
	// レスポンスヘッダーをデバッグ表示
	fmt.Printf("auth1レスポンスヘッダー:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	
	if resp.StatusCode != 200 {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		fmt.Printf("auth1エラーレスポンス: %s\n", buf.String())
		return fmt.Errorf("auth1認証失敗: HTTP %d", resp.StatusCode)
	}
	
	// 認証トークンを取得
	authToken := resp.Header.Get("X-Radiko-Authtoken")
	if authToken == "" {
		authToken = resp.Header.Get("x-radiko-authtoken")
	}
	if authToken == "" {
		fmt.Printf("認証トークンが見つかりませんでした\n")
		return fmt.Errorf("認証トークンが取得できませんでした")
	}
	
	// キー情報を取得
	keyLength := resp.Header.Get("X-Radiko-KeyLength")
	keyOffset := resp.Header.Get("X-Radiko-KeyOffset")
	if keyLength == "" {
		keyLength = resp.Header.Get("x-radiko-keylength")
	}
	if keyOffset == "" {
		keyOffset = resp.Header.Get("x-radiko-keyoffset")
	}
	
	fmt.Printf("認証トークン: %s... (長さ: %d)\n", authToken[:min(10, len(authToken))], len(authToken))
	fmt.Printf("キー長: %s, キーオフセット: %s\n", keyLength, keyOffset)
	
	// 部分鍵を生成
	partialKey, err := c.generatePartialKey(keyLength, keyOffset)
	if err != nil {
		return fmt.Errorf("部分鍵生成エラー: %w", err)
	}
	
	fmt.Printf("部分鍵生成完了: %s... (長さ: %d)\n", partialKey[:min(10, len(partialKey))], len(partialKey))
	
	// Step 2: auth2 - 認証の有効化
	auth2URL := "https://radiko.jp/v2/api/auth2"
	fmt.Printf("auth2 URL: %s\n", auth2URL)
	
	req2, err := http.NewRequest("POST", auth2URL, nil)
	if err != nil {
		return fmt.Errorf("auth2リクエスト作成エラー: %w", err)
	}
	
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req2.Header.Set("Accept", "*/*")
	req2.Header.Set("X-Radiko-AuthToken", authToken)
	req2.Header.Set("X-Radiko-Partialkey", partialKey)
	
	fmt.Printf("auth2リクエスト送信中...\n")
	fmt.Printf("使用する認証トークン: %s...\n", authToken[:min(10, len(authToken))])
	fmt.Printf("使用する部分鍵: %s...\n", partialKey[:min(10, len(partialKey))])
	
	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		fmt.Printf("auth2リクエストエラー: %v\n", err)
		return fmt.Errorf("auth2リクエストエラー: %w", err)
	}
	defer resp2.Body.Close()
	
	fmt.Printf("auth2レスポンス: HTTP %d\n", resp2.StatusCode)
	
	// auth2レスポンスヘッダーをデバッグ表示
	fmt.Printf("auth2レスポンスヘッダー:\n")
	for key, values := range resp2.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	
	// auth2レスポンス本文を読み取り
	var buf2 bytes.Buffer
	buf2.ReadFrom(resp2.Body)
	auth2Response := buf2.String()
	fmt.Printf("auth2レスポンス本文: %s\n", auth2Response)
	
	if resp2.StatusCode != 200 {
		return fmt.Errorf("auth2認証失敗: HTTP %d", resp2.StatusCode)
	}
	
	// 認証トークンを保存
	c.authToken = authToken
	
	c.logger.Info("radiko認証完了")
	fmt.Printf("=== RADIKO認証デバッグ完了 ===\n\n")
	return nil
}

// generatePartialKey は部分鍵を生成
func (c *Client) generatePartialKey(keyLengthStr, keyOffsetStr string) (string, error) {
	// radikoの共通鍵（公開されている情報）
	// 注意: この値は実際のradikoクライアントから取得される必要があります
	authKey := "bcd151073c03b352e1ef2fd66c32209da9ca0afa"
	
	keyLength, err := strconv.Atoi(keyLengthStr)
	if err != nil {
		return "", fmt.Errorf("キー長の変換エラー: %w", err)
	}
	
	keyOffset, err := strconv.Atoi(keyOffsetStr)
	if err != nil {
		return "", fmt.Errorf("キーオフセットの変換エラー: %w", err)
	}
	
	fmt.Printf("共通鍵: %s\n", authKey)
	fmt.Printf("抽出範囲: オフセット=%d, 長さ=%d\n", keyOffset, keyLength)
	
	// オフセットと長さが有効な範囲内かチェック
	if keyOffset < 0 || keyLength <= 0 || keyOffset+keyLength > len(authKey) {
		return "", fmt.Errorf("無効なキー範囲: offset=%d, length=%d, authKey length=%d", keyOffset, keyLength, len(authKey))
	}
	
	// 部分鍵を抽出
	partialKeyBytes := authKey[keyOffset : keyOffset+keyLength]
	fmt.Printf("抽出された部分鍵（hex）: %s\n", partialKeyBytes)
	
	// Base64エンコード
	partialKey := base64.StdEncoding.EncodeToString([]byte(partialKeyBytes))
	
	fmt.Printf("Base64エンコード後: %s\n", partialKey)
	
	return partialKey, nil
}

// min は2つの整数の最小値を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	fmt.Printf("=== ストリーミングURL取得デバッグ開始 ===\n")
	fmt.Printf("局ID: %s\n", stationID)
	
	// 認証が必要
	if c.authToken == "" {
		fmt.Printf("認証トークンが設定されていません\n")
		return "", fmt.Errorf("認証が必要です。先にAuth()を実行してください")
	}
	
	fmt.Printf("使用する認証トークン: %s... (長さ: %d)\n", c.authToken[:10], len(c.authToken))
	
	// ライブストリーミング用のプレイリストURL取得
	// radikoの正式なライブストリームエンドポイント
	streamInfoURL := fmt.Sprintf("https://radiko.jp/v2/api/ts/playlist.m3u8?station_id=%s&l=15&lsid=%d&type=b", 
		stationID, time.Now().Unix())
	
	fmt.Printf("ストリーミング情報URL: %s\n", streamInfoURL)
	
	req, err := http.NewRequest("GET", streamInfoURL, nil)
	if err != nil {
		fmt.Printf("リクエスト作成エラー: %v\n", err)
		return "", fmt.Errorf("ストリーミング情報リクエスト作成エラー: %w", err)
	}
	
	// 必要なヘッダーを設定（認証トークンを含む）
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("X-Radiko-AuthToken", c.authToken)
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	
	fmt.Printf("送信ヘッダー:\n")
	for key, values := range req.Header {
		for _, value := range values {
			if key == "X-Radiko-AuthToken" {
				fmt.Printf("  %s: %s... (長さ: %d)\n", key, value[:10], len(value))
			} else {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}
	
	c.logger.Debug("ストリーミング情報リクエスト送信中: %s", streamInfoURL)
	fmt.Printf("ストリーミング情報リクエスト送信中...\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("リクエストエラー: %v\n", err)
		return "", fmt.Errorf("ストリーミング情報リクエストエラー: %w", err)
	}
	defer resp.Body.Close()
	
	c.logger.Debug("ストリーミングレスポンス: HTTP %d", resp.StatusCode)
	fmt.Printf("ストリーミングレスポンス: HTTP %d\n", resp.StatusCode)
	
	// レスポンスヘッダーをデバッグ表示
	fmt.Printf("レスポンスヘッダー:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
	
	if resp.StatusCode != 200 {
		// エラーレスポンスの内容を表示
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		c.logger.Debug("ストリーミングエラーレスポンス: %s", buf.String())
		fmt.Printf("ストリーミングエラーレスポンス: %s\n", buf.String())
		return "", fmt.Errorf("ストリーミング情報取得失敗: HTTP %d", resp.StatusCode)
	}
	
	// レスポンスを読み取り（m3u8プレイリスト）
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		fmt.Printf("レスポンス読み取りエラー: %v\n", err)
		return "", fmt.Errorf("ストリーミング情報読み取りエラー: %w", err)
	}
	
	playlistContent := buf.String()
	c.logger.Debug("プレイリスト取得完了、内容長: %d bytes", len(playlistContent))
	fmt.Printf("プレイリスト取得完了、内容長: %d bytes\n", len(playlistContent))
	
	// プレイリスト内容の表示用に安全な長さを計算
	displayLength := len(playlistContent)
	if displayLength > 200 {
		displayLength = 200
	}
	c.logger.Debug("プレイリスト内容: %s", playlistContent[:displayLength]+"...")
	fmt.Printf("プレイリスト内容（最初の200文字）:\n%s\n", playlistContent[:displayLength])
	if len(playlistContent) > 200 {
		fmt.Printf("... (省略)\n")
	}
	
	// プレイリストから実際のストリーミングURLを抽出
	lines := strings.Split(playlistContent, "\n")
	fmt.Printf("プレイリスト解析中...\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		fmt.Printf("行 %d: %s\n", i+1, line)
		if strings.HasPrefix(line, "https://") && strings.Contains(line, ".m3u8") {
			c.logger.Debug("ストリーミングURL見つかりました: %s", line)
			fmt.Printf("ストリーミングURL発見: %s\n", line)
			fmt.Printf("=== ストリーミングURL取得デバッグ完了 ===\n\n")
			return line, nil
		}
	}
	
	// プレイリストに直接URLが含まれていない場合、元のURLを返す
	c.logger.Debug("直接プレイリストURLを使用: %s", streamInfoURL)
	fmt.Printf("直接URLが見つからないため、プレイリストURLを使用: %s\n", streamInfoURL)
	fmt.Printf("=== ストリーミングURL取得デバッグ完了 ===\n\n")
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
