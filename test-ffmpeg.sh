#!/bin/bash

echo "=== FFmpeg検出テスト ==="

# 通常のパス検索
echo "1. PATH内での検索:"
which ffmpeg

# 標準的なLinuxパス
echo "2. 標準的なLinuxパス:"
ls -la /usr/bin/ffmpeg 2>/dev/null || echo "見つかりません"

# カスタムインストールパス
echo "3. カスタムインストールパス:"
ls -la /usr/local/bin/ffmpeg 2>/dev/null || echo "見つかりません"

# macOS Homebrew (Apple Silicon)
echo "4. macOS Homebrew (Apple Silicon):"
ls -la /opt/homebrew/bin/ffmpeg 2>/dev/null || echo "見つかりません"

# 環境変数
echo "5. 環境変数 FFMPEG_PATH:"
echo "${FFMPEG_PATH:-未設定}"

echo "=== テスト完了 ==="
