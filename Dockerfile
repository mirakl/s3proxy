FROM golang:1.10.2-stretch as app-builder

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


FROM golang:1.10.2-stretch as lib-builder

WORKDIR /root
RUN git clone https://github.com/mirakl/dns-aaaa-no-more.git && \
    cd dns-aaaa-no-more && \
    make


FROM centos:latest

COPY --from=lib-builder /root/dns-aaaa-no-more/getaddrinfo.so /dns-aaaa-no-more/
COPY --from=app-builder /go/src/github.com/mirakl/s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody

# To fix DNS issues in K8S caused by conntrack race condition (A/AAAA sent in parallel):
# - cgo resolver is enforced (see https://golang.org/pkg/net/#hdr-Name_Resolution)
# - getaddrinfo() C function called by cgo resolver is hooked to a new one not sending AAAA DNS requests
ENTRYPOINT GODEBUG=netdns=cgo LD_PRELOAD=/dns-aaaa-no-more/getaddrinfo.so exec /bin/s3proxy
