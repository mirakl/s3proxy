FROM golang:1.12-stretch as app-builder

ARG VERSION

ENV SRC_DIR /s3proxy
WORKDIR $SRC_DIR

RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.17.1

ENV GO111MODULE on

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN make VERSION=${VERSION}

FROM golang:1.12-stretch as lib-builder

WORKDIR /root
RUN git clone https://github.com/mirakl/dns-aaaa-no-more.git && \
    cd dns-aaaa-no-more && \
    make


FROM centos:latest

COPY --from=lib-builder /root/dns-aaaa-no-more/getaddrinfo.so /dns-aaaa-no-more/
COPY --from=app-builder /s3proxy /bin
RUN chmod +x /bin/s3proxy

EXPOSE 8080

USER nobody

# To fix DNS issues in K8S caused by conntrack race condition (A/AAAA sent in parallel):
# - cgo resolver is enforced (see https://golang.org/pkg/net/#hdr-Name_Resolution)
# - getaddrinfo() C function called by cgo resolver is hooked to a new one not sending AAAA DNS requests
ENTRYPOINT GODEBUG=netdns=cgo LD_PRELOAD=/dns-aaaa-no-more/getaddrinfo.so exec /bin/s3proxy
