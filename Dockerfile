FROM golang:1.12-alpine3.10 as build

RUN apk upgrade && \
  apk update && \
  apk add --update git ca-certificates

WORKDIR $GOPATH/src/github.com/evilwire/flex-sftp

ENV GO111MODULE on
ADD go.mod go.sum ./
RUN go mod download

ADD . ./
ENV CGO_ENABLED 0
RUN go get && \
    go build -ldflags '-w' \
             -o $GOPATH/bin/flex-sftp

FROM alpine:3.10

ARG VERSION
ENV VERSION $VERSION

ARG SHORTHASH
ENV SHORTHASH $SHORTHASH

ARG BUILD_TIME
ENV BUILD_TIME $BUILD_TIME

COPY --from=build /go/bin /bin

VOLUME /usr/keys
EXPOSE 2022

ENTRYPOINT /bin/flex-sftp