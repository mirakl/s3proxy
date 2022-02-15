FROM golang:1.16-alpine as app-builder

ENV SRC_DIR /s3proxy
WORKDIR $SRC_DIR

ENV GO111MODULE on

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN apk add build-base curl && make


FROM golang:1.16-alpine as lib-builder

WORKDIR /root
RUN apk add git
RUN git clone https://github.com/mirakl/dns-aaaa-no-more.git
RUN apk add build-base && cd dns-aaaa-no-more && make


FROM alpine:3.15

# Magic !
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY --from=lib-builder /root/dns-aaaa-no-more/getaddrinfo.so /dns-aaaa-no-more/
COPY --from=app-builder /s3proxy/s3proxy /usr/local/bin/
RUN chmod +x /usr/local/bin/s3proxy

EXPOSE 8080

USER nobody

# To fix DNS issues in K8S caused by conntrack race condition (A/AAAA sent in parallel):
# - cgo resolver is enforced (see https://golang.org/pkg/net/#hdr-Name_Resolution)
# - getaddrinfo() C function called by cgo resolver is hooked to a new one not sending AAAA DNS requests
ENTRYPOINT GODEBUG=netdns=cgo LD_PRELOAD=/dns-aaaa-no-more/getaddrinfo.so exec /usr/local/bin/s3proxy
