.PHONY: clean,all

all: bin/agent bin/server bin/health
	
bin/agent: $(shell find . -name '*.go') proto/service.pb.go
	go build -o bin/agent cmd/agent/*.go

bin/server: $(shell find . -name '*.go') proto/service.pb.go
	go build -o bin/server cmd/server/*.go

bin/health: $(shell find . -name '*.go') proto/service.pb.go
	go build -o bin/health cmd/health/*.go

proto/service.pb.go: proto/service.proto
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	protoc -I proto/ proto/service.proto --go_out=plugins=grpc:proto

clean:
	rm -rf bin/