.PHONY: test clean all

all: bin/kiam

bin/kiam: $(shell find . -name '*.go') proto/service.pb.go
	go build -o bin/kiam cmd/kiam/*.go

proto/service.pb.go: proto/service.proto
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	protoc -I proto/ proto/service.proto --go_out=plugins=grpc:proto

test: $(shell find . -name '*.go')
	go test test/unit/*_test.go
	go test test/functional/*_test.go

clean:
	rm -rf bin/
