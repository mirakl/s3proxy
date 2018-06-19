FROM golang:1.10.2-stretch as builder

ARG VERSION

RUN go get -u github.com/golang/dep/cmd/dep
RUN go get -u gopkg.in/alecthomas/gometalinter.v2
RUN gometalinter.v2 --install

ENV SRC_DIR /go/src/github.com/mirakl/s3proxy
WORKDIR $SRC_DIR

COPY backend/ ./backend
COPY logger/ ./logger
COPY middleware/ ./middleware
COPY router/ ./router
COPY s3proxytest/ ./s3proxytest
COPY util/ ./util
COPY Makefile ./
COPY *.go ./

RUN make VERSION=${VERSION}

FROM centos:latest

COPY --from=builder /go/src/github.com/mirakl/s3proxy/s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody
ENTRYPOINT ["s3proxy"]
