# Alternative Dockerfile with static ffmpeg binary
# Build stage
FROM golang:1.24.2 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main lambda/handler.go

# ffmpeg static binary download stage
FROM alpine:latest AS ffmpeg-builder
RUN apk add --no-cache curl
WORKDIR /tmp
RUN curl -L https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz \
    -o ffmpeg.tar.xz && \
    tar xf ffmpeg.tar.xz && \
    mv ffmpeg-master-latest-linux64-gpl/bin/ffmpeg /usr/local/bin/ffmpeg && \
    chmod +x /usr/local/bin/ffmpeg

# Runtime stage
FROM public.ecr.aws/lambda/provided:al2023

# Copy static ffmpeg binary
COPY --from=ffmpeg-builder /usr/local/bin/ffmpeg /usr/local/bin/ffmpeg

# Goアプリケーションをコピー
COPY --from=builder /src/main ${LAMBDA_TASK_ROOT}/bootstrap
RUN chmod +x ${LAMBDA_TASK_ROOT}/bootstrap

# ffmpegのパスを環境変数に設定
ENV PATH="/usr/local/bin:${PATH}"

CMD ["bootstrap"]
