FROM golang:1.9.4-stretch as builder

RUN go get -u github.com/golang/dep/cmd/dep

ENV SRC_DIR /go/src/s3proxy
WORKDIR $SRC_DIR

COPY Gopkg.* ./
COPY vendor/ ./vendor
COPY middleware/ ./middleware
COPY logging/ ./logging
COPY Makefile ./
COPY *.go ./

RUN make

FROM centos:latest

COPY --from=builder /go/src/s3proxy/s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody
ENTRYPOINT ["s3proxy"]
