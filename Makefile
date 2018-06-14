NAME = s3proxy
REMOTE_NAME = ${REGISTRY}/${NAME}

GOPATH ?= ${HOME}/go
VERSION ?= 1.2.0

LDFLAGS=-ldflags "-X main.version=${VERSION}"

GOFILES	= $(shell find . -type f -name '*.go' -not -path "./vendor/*")

default: build

build: clean fmtcheck dep lint
	go build -i -v ${LDFLAGS} -o ${NAME}

dep:
	if [ -f "Gopkg.toml" ] ; then dep ensure ; else dep init ; fi

clean:
	if [ -f "${NAME}" ] ; then rm ${NAME} ; fi

lint:
	${GOPATH}/bin/gometalinter.v2 go --vendor --tests --errors --concurrency=2 --deadline=60s ./...

fmtcheck: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
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
.PHONY: test
