FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sensor-dump .

FROM alpine:3.20

RUN apk add --no-cache postgresql16-client
WORKDIR /work
COPY --from=builder /out/sensor-dump /usr/local/bin/sensor-dump
ENTRYPOINT ["/usr/local/bin/sensor-dump"]
