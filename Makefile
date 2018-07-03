.PHONY: test clean all

all: bin/kiam

bin/kiam: $(shell find . -name '*.go') proto/service.pb.go
	go build -o bin/kiam cmd/kiam/*.go

proto/service.pb.go: proto/service.proto
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	protoc -I proto/ proto/service.proto --go_out=plugins=grpc:proto

test: $(shell find . -name '*.go')
	go test github.com/uswitch/kiam/pkg/... -cover
	go test test/functional/*_test.go

coverage: $(shell find . -name '*.go')
	go test github.com/uswitch/kiam/pkg/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	rm -rf bin/
