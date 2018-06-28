ARCH=amd64
BIN = bin/kiam
BIN_LINUX = $(BIN)-linux-$(ARCH)
BIN_DARWIN = $(BIN)-darwin-$(ARCH)

SOURCES := $(shell find . -not -path '*vendor*' -iname '*.go') proto/service.pb.go

.PHONY: test clean all

all: proto/service.pb.go build-darwin build-linux

build-darwin: $(SOURCES)
	GOARCH=$(ARCH) GOOS=darwin go build -o $(BIN_DARWIN) cmd/kiam/*.go

build-linux: $(SOURCES)
	GOARCH=$(ARCH) GOOS=linux CGO_ENABLED=0 go build -o $(BIN_LINUX) cmd/kiam/*.go

proto/service.pb.go: proto/service.proto
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	protoc -I proto/ proto/service.proto --go_out=plugins=grpc:proto

test: $(SOURCES)
	go test test/unit/*_test.go
	go test test/functional/*_test.go

bench: $(SOURCES)
	go test -run=XX -bench=. test/unit/*.go

docker: Dockerfile $(BIN_LINUX)
	docker image build -t quay.io/uswitch/kiam:devel .

clean:
	rm -rf bin/
