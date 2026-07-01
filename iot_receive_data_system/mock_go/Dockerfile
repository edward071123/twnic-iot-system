FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mock_sender ./main.go

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/mock_sender /usr/local/bin/mock_sender

ENTRYPOINT ["mock_sender"]
