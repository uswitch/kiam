NAME?=kiam
BIN = bin/kiam
BIN_LINUX = $(BIN)-linux
GIT_BRANCH?=$(shell git rev-parse --abbrev-ref HEAD)
IMG_NAMESPACE?=quay.io/uswitch
IMG_TAG?=$(GIT_BRANCH)
GO111MODULE=on
REGISTRY?=$(IMG_NAMESPACE)/$(NAME)
SOURCES := $(shell find . -iname '*.go') proto/service.pb.go

.PHONY: test clean all coverage

all: $(BIN_LINUX)

$(BIN_LINUX): $(SOURCES)
	go build -o $(BIN_LINUX) cmd/kiam/*.go

proto/service.pb.go: proto/service.proto
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	protoc -I proto/ proto/service.proto --go_out=plugins=grpc:proto

test: $(SOURCES)
	go test github.com/uswitch/kiam/pkg/... -race

coverage.txt: $(SOURCES)
	go test github.com/uswitch/kiam/pkg/... -coverprofile=coverage.txt -covermode=atomic

coverage: $(SOURCES) coverage.txt
	go tool cover -html=coverage.txt

bench: $(SOURCES)
	go test -run=XX -bench=. github.com/uswitch/kiam/pkg/...

docker: Dockerfile
	docker image build -t "$(REGISTRY):$(IMG_TAG)" .

clean:
	rm -rf bin/
