FROM golang:alpine as build
MAINTAINER michael@kth.se
LABEL maintainer "michael@kth.se"
ENV LOGSPOUT_VERSION=3.2.6
RUN mkdir -p /go/src
WORKDIR /go/src
VOLUME /mnt/routes
EXPOSE 80

RUN apk --no-cache add --update curl git gcc musl-dev go build-base git mercurial ca-certificates
RUN curl -fSL -o logspout_v${LOGSPOUT_VERSION}.tar.gz "https://github.com/gliderlabs/logspout/archive/v${LOGSPOUT_VERSION}.tar.gz" \
    && tar -zxvf logspout_v${LOGSPOUT_VERSION}.tar.gz \
    && rm logspout_v${LOGSPOUT_VERSION}.tar.gz \
    && mkdir -p /go/src/github.com/gliderlabs/ \
    && mv logspout-${LOGSPOUT_VERSION} /go/src/github.com/gliderlabs/logspout

WORKDIR /go/src/github.com/gliderlabs/logspout
RUN echo 'import ( _ "github.com/gliderlabs/logspout/adapters/raw" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/adapters/syslog" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/httpstream" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/routesapi" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/transports/tcp" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/transports/udp" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/transports/tls" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/healthcheck" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/gliderlabs/logspout/adapters/multiline" )' >> /go/src/github.com/gliderlabs/logspout/modules.go \
    && echo 'import ( _ "github.com/mictsi/logspout-gelf-1" )' >> /go/src/github.com/gliderlabs/logspout/modules.go

RUN go get -d -v ./...
RUN go build -v -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/logspout

FROM alpine:latest
RUN apk --no-cache add --update ca-certificates
COPY --from=build /go/src/github.com/gliderlabs/logspout/bin/logspout /go/bin/logspout
ENTRYPOINT ["/go/bin/logspout"]
