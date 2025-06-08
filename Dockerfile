# AWS Lambda用 Go-Radio Dockerfile (ffmpeg静的バイナリ版)
# Build stage
FROM golang:1.24.2 AS builder
WORKDIR /src

# Go modulesをコピーして依存関係をダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# ファイル構造を確認
RUN echo "=== ファイル構造確認 ===" && \
    ls -la && \
    echo "=== lambda/ ディレクトリ ===" && \
    ls -la lambda/ && \
    echo "=== handler.go 存在確認 ===" && \
    test -f lambda/handler.go && echo "handler.go が存在します" || echo "handler.go が見つかりません"

# Goアプリケーションをビルド
RUN echo "=== Goビルド開始 ===" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main lambda/handler.go && \
    echo "=== ビルド完了 ===" && \
    ls -la main && \
    echo "=== ファイルサイズ確認 ===" && \
    du -h main

# Runtime stage
FROM public.ecr.aws/lambda/provided:al2023

# # 静的ffmpegバイナリをダウンロード
# RUN dnf update -y && \
#     dnf install -y tar xz && \
#     dnf clean all

# # John Van Sickleの静的ビルドを使用（最も確実）
# RUN curl -L -o /tmp/ffmpeg.tar.xz \
#         https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz && \
#     tar -C /opt -xJf /tmp/ffmpeg.tar.xz && \
#     mv /opt/ffmpeg-*-static /opt/ffmpeg && \
#     ln -s /opt/ffmpeg/ffmpeg  /usr/local/bin/ffmpeg  && \
#     ln -s /opt/ffmpeg/ffprobe /usr/local/bin/ffprobe && \
#     rm -f /tmp/ffmpeg.tar.xz && \
#     chmod +x /usr/local/bin/ffmpeg /usr/local/bin/ffprobe

# Goアプリケーションをコピー
COPY --from=builder /src/main ${LAMBDA_TASK_ROOT}/bootstrap

# 実行権限を設定し、ランタイムディレクトリにもシンボリックリンクを作成
RUN chmod +x ${LAMBDA_TASK_ROOT}/bootstrap && \
    mkdir -p /var/runtime && \
    ln -sf ${LAMBDA_TASK_ROOT}/bootstrap /var/runtime/bootstrap

# コピーされたファイルを確認

# ffmpegのパスを環境変数に設定
# ENV PATH="/usr/local/bin:${PATH}"
# ENV FFMPEG_PATH="/usr/local/bin/ffmpeg"

# ffmpegのテスト
# RUN /usr/local/bin/ffmpeg -version

CMD ["bootstrap"]
