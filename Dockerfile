FROM golang:1.15.15 as build
ENV GO111MODULE=on

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
COPY proto/ proto/
COPY Makefile Makefile

# As (generated) proto/service.pb.go is _also_ committed, void the need to install protoc / protoc-gen-go plugin
RUN touch proto/service.pb.go
RUN CGO_ENABLED=0 go build -o bin/kiam cmd/kiam/*.go

FROM alpine:3.11
RUN apk --no-cache add iptables
COPY --from=build /workspace/bin/kiam /kiam
CMD []
