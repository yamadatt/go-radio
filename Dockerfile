# Build stage
FROM golang:1.24.2 AS builder
WORKDIR /src
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main lambda/handler.go

# Runtime stage
FROM public.ecr.aws/lambda/go:1
COPY --from=builder /src/main ${LAMBDA_TASK_ROOT}/bootstrap
CMD ["bootstrap"]
