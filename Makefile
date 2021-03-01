VERSION ?= $(shell git describe --match 'v[0-9]*' --tags --always)

lint:
	@go fmt ./... && go vet ./...

test:
	@go test -v ./... -coverprofile cover.out

build: clean
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/kubectl-mc

generate:
	go generate ./...

install:
	@go install -ldflags "-X main.version=$(VERSION)"

clean:
	@ rm -rf bin dist
