VERSION ?= 1.0.0
GOPATH ?= ${HOME}/go

BINARY = s3proxy
PKG = s3proxy
PKG_LIST := $(go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(find . -name '*.go' | grep -v /vendor/)

LDFLAGS=-ldflags "-X main.Version=${VERSION}"

default: build

build: clean deps vet lint
	go build -i -v ${LDFLAGS} -o ${BINARY}

deps:
	dep ensure

clean:
	if [ -f "${BINARY}" ] ; then rm ${BINARY} ; fi

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		go lint $$file ; \
	done

fmt:
	@for file in ${GO_FILES} ;  do \
		go fmt $$file ; \
	done

docker-image:
	docker build . -t mirakl/s3proxy
