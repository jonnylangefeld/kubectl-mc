VERSION ?= $(shell git describe --match 'v[0-9]*' --tags --always)

lint:
	@go fmt ./... && go vet ./...

build: clean
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/kubectl-mc

install:
	@go install -ldflags "-X main.version=$(VERSION)"

clean:
	@ rm -rf bin dist