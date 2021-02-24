FROM golang:1.15

ENV GO111MODULE=on \
    GOOS=linux \
    GARCH=amd64 \
    CGO_ENABLED=0

WORKDIR /usr/src/app

RUN go get -v github.com/spf13/cobra/cobra

