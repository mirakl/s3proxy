NAME = s3proxy
REGISTRY = eu.gcr.io/mirakl-production/kube/mp
REMOTE_NAME = ${REGISTRY}/${NAME}

GOPATH ?= ${HOME}/go
VERSION ?= 1.1.0

LDFLAGS=-ldflags "-X main.version=${VERSION}"

default: build

build: clean fmt dep lint
	go build -i -v ${LDFLAGS} -o ${NAME}

dep:
	if [ -f "Gopkg.toml" ] ; then dep ensure ; else dep init ; fi

clean:
	if [ -f "${NAME}" ] ; then rm ${NAME} ; fi

lint:
	${GOPATH}/bin/gometalinter.v2 go --vendor --tests --errors --concurrency=2 --deadline=60s ./...

fmt:
	go fmt ./...

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
	docker build . -t mirakl/${NAME}:${VERSION} -t ${REMOTE_NAME}:${VERSION} --build-arg VERSION=${VERSION}

docker-image-push: docker-image
	docker push ${REMOTE_NAME}:${VERSION}

check-version:
ifndef VERSION
	$(error VERSION is undefined)
endif

.PHONY: test
