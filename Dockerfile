FROM registry.js.design/library/golang:1.23.3-otel-amd64 AS builder

ARG VERSION

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /root

COPY . /root

RUN CGO_ENABLED=0 go build -o sreflow . \
    && chmod +x sreflow

FROM alpine:3.19

COPY --from=builder /root/sreflow /app/sreflow

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories \
    && apk upgrade && apk add --no-cache --virtual .build-deps \
    ca-certificates upx tzdata

WORKDIR /app

ENTRYPOINT ["/app/sreflow"]
