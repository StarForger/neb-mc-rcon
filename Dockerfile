FROM golang:1.15

ENV GO111MODULE=on

WORKDIR /usr/src/app

RUN go get -v github.com/spf13/cobra/cobra

