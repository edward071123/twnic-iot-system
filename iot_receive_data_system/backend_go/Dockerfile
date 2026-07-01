# Build stage: compile static Linux binary.
# 建置階段：編譯靜態 Linux 執行檔。
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend_go ./cmd

# Runtime stage: small image with only compiled binary.
# 執行階段：只保留編譯後二進位檔的輕量映像。
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache postgresql16-client
COPY --from=builder /out/backend_go /usr/local/bin/backend_go
EXPOSE 3002
ENTRYPOINT ["backend_go"]
