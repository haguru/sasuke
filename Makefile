.PHONY: all privatekey tidy build clean lint unittest test
.PHONY: fmt run install-lint pull-mongo start-mongo stop-mongo 
.PHONY: pull-postgres start-postgres stop-all-containers

all: privatekey build

ARCH=$(shell uname -m)
MICROSERVICE=sasuke
KEYPATH=./res/sharingan_key.pem

privatekey:
	openssl ecparam -name prime256v1 -genkey -noout -out $(KEYPATH)

fmt:
	go fmt ./...

tidy:
	go mod tidy

unittest:
	go test ./... -coverprofile=coverage.out ./...

test: unittest lint
	go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]

build:
	go build -o sasuke

run: build
	./$(MICROSERVICE)

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run make install-lint"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

install-lint:
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2

clean:
	rm -f sharingan_key.pem sasuke

pull-mongo:
	docker pull mongodb/mongodb-community-server:latest

start-mongo:
	docker run --name sasuke-mongo -d -p 27017:27017 mongodb/mongodb-community-server:latest

pull-postgres:
	docker pull postgres:latest

start-postgres:
	docker run --name sasuke-postgres -d -p 5432:5432 postgres:latest

stop-all-containers:
	docker stop sasuke-mongo && docker rm sasuke-mongo
	docker stop sasuke-postgres && docker rm sasuke-postgres