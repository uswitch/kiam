FROM golang:1.15.7 as build

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

RUN make bin/kiam-linux-amd64

FROM alpine:3.13

COPY --from=build /workspace/bin/kiam-linux-amd64 /usr/local/bin/kiam

RUN apk --no-cache add \
    ca-certificates \
    iptables \
    && update-ca-certificates

ENTRYPOINT ["kiam"]
