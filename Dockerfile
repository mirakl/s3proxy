FROM golang:1.9.4-stretch as builder

ENV GOPATH=/go

RUN go get -u github.com/golang/dep/cmd/dep

ENV SRC_DIR /go/src/s3proxy
WORKDIR $SRC_DIR

COPY Gopkg.* $SRC_DIR/
COPY vendor/ $SRC_DIR/vendor
COPY middleware/ $SRC_DIR/middleware
COPY logging/ $SRC_DIR/logging
COPY Makefile $SRC_DIR/
COPY *.go $SRC_DIR/

RUN make

FROM centos:latest

COPY --from=builder /go/src/s3proxy/s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody
ENTRYPOINT ["s3proxy"]
