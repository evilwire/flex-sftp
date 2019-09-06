FROM golang:1.12-alpine3.10 as build

RUN apk upgrade && \
  apk update && \
  apk add --update git ca-certificates

WORKDIR $GOPATH/src/github.com/evilwire/flex-sftp

ENV GO111MODULE on
ADD go.mod go.sum ./
RUN go mod download

ADD . ./
RUN go get && \
    go build -a \
             -tags netgo \
             -ldflags '-w' \
             -o $GOPATH/bin/flex-sftp

FROM alpine:3.10

COPY --from=build /go/bin /bin

VOLUME /usr/keys
EXPOSE 2022

ENTRYPOINT /bin/flex-sftp