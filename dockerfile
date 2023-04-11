# goバージョン
FROM golang:1.17.13 AS builder
# appディレクトリの作成
RUN mkdir /go/src/app
# ワーキングディレクトリの設定
WORKDIR /go/src/app
# ホストのファイルをコンテナの作業ディレクトリに移行
ADD . /go/src/app
# バイナリ作成
RUN  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build go_tc
# バイナリをalpine上で起動
FROM docker:latest 
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/app/go_tc ./
ENTRYPOINT ["./go_tc"]
