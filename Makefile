NAME = s3proxy
REMOTE_NAME = ${REGISTRY}${NAME}

GOPATH ?= ${HOME}/go
VERSION ?= 1.2.1

LDFLAGS=-ldflags "-X main.version=${VERSION}"

GOFILES	= $(shell find . -type f -name '*.go' -not -path "./vendor/*")

default: build

build: clean dep fmtcheck lint
	go build -i -v ${LDFLAGS} -o ${NAME}

dep: tools.dep
	if [ -f "Gopkg.toml" ] ; then dep ensure ; else dep init ; fi

clean:
	if [ -f "${NAME}" ] ; then rm ${NAME} ; fi

lint: tools.gometalinter.v2
	${GOPATH}/bin/gometalinter.v2 go --vendor --tests --errors --concurrency=2 --deadline=60s ./...

fmtcheck: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -d $(GOFILES)
	@! goimports -l $(GOFILES) | grep -vF 'Nope nope nope'

fmt: tools.goimports
	goimports -w $(GOFILES)

test:
	go test -v ./...

integration-test:
	docker-compose -f ./test/docker-compose.yml up -d minio rsyslog createbuckets
	docker run --rm --net=s3proxy-network -v ${GOPATH}:/go -i golang go test -v github.com/mirakl/s3proxy/test/... -tags=integration
	docker-compose -f ./test/docker-compose.yml down

end2end-test: docker-image
	VERSION=${VERSION} docker-compose -f ./test/docker-compose.yml up -d
	docker run --rm --net=s3proxy-network -v ${GOPATH}:/go -i golang go test -v github.com/mirakl/s3proxy/test/... -tags=end2end
	VERSION=${VERSION} docker-compose -f ./test/docker-compose.yml down

docker-image: check-version
	docker build . -t mirakl/${NAME}:${VERSION} -t ${REMOTE_NAME}:${VERSION} -t ${REMOTE_NAME}:latest --build-arg VERSION=${VERSION}

docker-image-push: docker-image
	docker push ${REMOTE_NAME}:${VERSION}
	docker push ${REMOTE_NAME}:latest

check-version:
ifndef VERSION
	$(error VERSION is undefined)
endif

tools.goimports:
	@command -v goimports >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing goimports"; \
		go get golang.org/x/tools/cmd/goimports; \
	fi

tools.dep:
	@command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing dep"; \
		go get github.com/golang/dep/cmd/dep; \
	fi

tools.gometalinter.v2:
	@command -v gometalinter.v2 >/dev/null ; if [ $$? -ne 0 ]; then \
		echo "--> installing gometalinter.v2"; \
		go get gopkg.in/alecthomas/gometalinter.v2; \
		gometalinter.v2 --install; \
	fi
.PHONY: test
