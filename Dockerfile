FROM golang:1.10.3-alpine
RUN apk add make && \
    mkdir -p /go/src/github.com/talon-one/talon-access-proxy
WORKDIR /go/src/github.com/talon-one/talon-access-proxy
