package radiko

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client はradikoクライアント
type Client struct {
	authToken  string
	areaID     string
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
		"TBS":   "TBSラジオ",
		"LFR":   "ニッポン放送",
		"QRR":   "文化放送",
		"RN1":   "ラジオNIKKEI第1",
		"RN2":   "ラジオNIKKEI第2",
		"INT":   "interfm",
		"FMT":   "TOKYO FM",
		"FMJ":   "J-WAVE",
		"JORF":  "ラジオ日本",
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

	req, err := http.NewRequest("GET", auth1URL, nil)
	if err != nil {
		return fmt.Errorf("auth1リクエスト作成エラー: %w", err)
	}

	// 必要なヘッダーを設定
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("X-Radiko-App", "pc_html5")
	req.Header.Set("X-Radiko-App-Version", "0.0.1")
	req.Header.Set("X-Radiko-User", "dummy_user")
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

	req2, err := http.NewRequest("GET", auth2URL, nil)
	if err != nil {
		return fmt.Errorf("auth2リクエスト作成エラー: %w", err)
	}

	req2.Header.Set("User-Agent", "Mozilla/5.0")
	req2.Header.Set("Accept", "*/*")
	req2.Header.Set("Pragma", "no-cache")
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

	// レスポンスからエリアIDを取得
	parts := strings.FieldsFunc(auth2Response, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	if len(parts) > 0 {
		c.areaID = strings.TrimSpace(parts[0])
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

// RecordTimeFree はタイムフリー番組を録音する
func (c *Client) RecordTimeFree(stationID string, startTime time.Time, duration int, outputFile string) error {
	endTime := startTime.Add(time.Duration(duration) * time.Minute)

	c.logger.Debug("録音設定: 局=%s, 開始=%s, 終了=%s", stationID, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	// タイムフリー再生用URLを取得
	streamURL, err := c.getTimeFreeURL(stationID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("ストリーミングURL取得エラー: %w", err)
	}

	c.logger.Info("ストリーミングURL取得完了")
	c.logger.Debug("ストリーミングURL: %s", streamURL)
	c.logger.Info("ダウンロード開始...")

	// Goのみを用いて音声ファイルをダウンロード
	return c.downloadWithGo(streamURL, outputFile)
}

// getTimeFreeURL はタイムフリー再生用のプレイリストURLを取得
func (c *Client) getTimeFreeURL(stationID string, startTime, endTime time.Time) (string, error) {
	c.logger.Debug("ストリーミングURL取得を開始: 局=%s", stationID)
	fmt.Printf("=== ストリーミングURL取得デバッグ開始 ===\n")
	fmt.Printf("局ID: %s\n", stationID)
	if !startTime.IsZero() {
		fmt.Printf("開始: %s\n", startTime.Format("2006-01-02 15:04:05"))
	}
	if !endTime.IsZero() {
		fmt.Printf("終了: %s\n", endTime.Format("2006-01-02 15:04:05"))
	}

	// 認証が必要
	if c.authToken == "" {
		fmt.Printf("認証トークンが設定されていません\n")
		return "", fmt.Errorf("認証が必要です。先にAuth()を実行してください")
	}

	fmt.Printf("使用する認証トークン: %s... (長さ: %d)\n", c.authToken[:10], len(c.authToken))

	// タイムフリー用のプレイリストURLを構築
	streamInfoURL := fmt.Sprintf("https://radiko.jp/v2/api/ts/playlist.m3u8?station_id=%s", stationID)
	if !startTime.IsZero() {
		streamInfoURL += "&ft=" + startTime.Format("20060102150405")
	}
	if !endTime.IsZero() {
		streamInfoURL += "&to=" + endTime.Format("20060102150405")
	}

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

// downloadWithGo はffmpegを使用せずGoだけで音声をダウンロード
func (c *Client) downloadWithGo(streamURL, outputFile string) error {
	segments, err := c.fetchSegments(streamURL)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return fmt.Errorf("プレイリストからセグメントを取得できませんでした")
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	for i, segURL := range segments {
		req, err := http.NewRequest("GET", segURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		if c.authToken != "" {
			req.Header.Set("X-Radiko-AuthToken", c.authToken)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("セグメントダウンロード失敗: %s", resp.Status)
		}
		if _, err := io.Copy(out, resp.Body); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()
		if (i+1)%10 == 0 {
			c.logger.Info("%d/%dセグメント完了", i+1, len(segments))
		}
	}

	fi, err := out.Stat()
	if err == nil {
		c.logger.Info("ダウンロード完了: %s (%.2f MB)", outputFile, float64(fi.Size())/(1024*1024))
		return nil
	}
	return err
}

// fetchSegments はm3u8プレイリストを再帰的に解析し、セグメントURLを取得
func (c *Client) fetchSegments(playlistURL string) ([]string, error) {
	req, err := http.NewRequest("GET", playlistURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if c.authToken != "" {
		req.Header.Set("X-Radiko-AuthToken", c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("プレイリスト取得失敗: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	base := playlistURL[:strings.LastIndex(playlistURL, "/")+1]
	var segments []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u := line
		if !strings.HasPrefix(u, "http") {
			u = base + u
		}
		if strings.Contains(u, ".m3u8") {
			sub, err := c.fetchSegments(u)
			if err != nil {
				return nil, err
			}
			segments = append(segments, sub...)
		} else {
			segments = append(segments, u)
		}
	}
	return segments, nil
}
