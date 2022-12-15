# goバージョン
FROM golang:1.16.3-alpine
# アップデートとgitのインストール！！
RUN apk add --update &&  apk add git && apk add iproute2-tc && apk add bash
# appディレクトリの作成
RUN mkdir /go/src/app
# ワーキングディレクトリの設定
WORKDIR /go/src/app
# ホストのファイルをコンテナの作業ディレクトリに移行
ADD . /go/src/app
# バイナリ作成
RUN go build go_tc
# アプリ起動
ENTRYPOINT ["./go_tc"]

