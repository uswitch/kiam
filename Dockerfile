FROM golang:1.12-alpine3.10 as build
RUN apk add --no-cache make
WORKDIR /go/src/github.com/uswitch/kiam
ADD . .
RUN make bin/kiam-linux-amd64

FROM alpine:3.8
RUN apk --no-cache add iptables
COPY --from=build /go/src/github.com/uswitch/kiam/bin/kiam-linux-amd64 /kiam
CMD []
