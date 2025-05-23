NAME = s3proxy
REMOTE_NAME = ${REGISTRY}${NAME}

GOPATH ?= ${HOME}/go
VERSION ?= 1.4.5

LDFLAGS=-ldflags "-X main.version=${VERSION}"

GOFILES	= $(shell find . -type f -name '*.go' -not -path "./vendor/*")

default: build

build: clean
	go build -v ${LDFLAGS} -o ${NAME}

clean:
	if [ -f "${NAME}" ] ; then rm ${NAME} ; fi

lint:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v2.1.6 golangci-lint run -v

fmtcheck: tools.goimports
	@echo "--> checking code formatting with 'goimports' tool"
	@goimports -d $(GOFILES)
	@! goimports -l $(GOFILES) | grep -vF 'Nope nope nope'

fmt: tools.goimports
	goimports -w $(GOFILES)

test:
	go test -v ./...

integration-test: docker-build-image
	docker-compose -f ./test/docker-compose.yml up -d minio rsyslog createbuckets
	docker run --rm --net=s3proxy-network -i mirakl/${NAME}-build go test -v ./test -tags=integration
	docker-compose -f ./test/docker-compose.yml down

end2end-test: docker-build-image
	docker-compose -f ./test/docker-compose.yml up -d
	docker run --rm --net=s3proxy-network -i mirakl/${NAME}-build go test -v ./test -tags=end2end
	docker-compose -f ./test/docker-compose.yml down

docker-image: check-version
	docker build . -t mirakl/${NAME}:${VERSION} -t ${REMOTE_NAME}:${VERSION} -t ${REMOTE_NAME}:latest --build-arg VERSION=${VERSION}

docker-build-image:
	docker build . -t mirakl/${NAME}-build --target app-builder

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
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi

.PHONY: test
