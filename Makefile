.PHONY: all privatekey tidy build clean lint

all: privatekey build

ARCH=$(shell uname -m)
MICROSERVICE=sasuke

privatekey:
	openssl ecparam -name prime256v1 -genkey -noout -out ./res/sharingan_key.pem
	

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