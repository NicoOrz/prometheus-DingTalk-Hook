FROM golang:1.22 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/prometheus-dingtalk-hook ./cmd/prometheus-dingtalk-hook

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/prometheus-dingtalk-hook /app/prometheus-dingtalk-hook

ENTRYPOINT ["/app/prometheus-dingtalk-hook"]

