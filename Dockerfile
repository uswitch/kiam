FROM golang:1.11-alpine3.9 as build
RUN apk add --no-cache make
WORKDIR /go/src/github.com/uswitch/kiam
ADD . .
RUN make bin/kiam-linux-amd64

FROM alpine:3.9
RUN apk --no-cache add iptables
COPY --from=build /go/src/github.com/uswitch/kiam/bin/kiam-linux-amd64 /kiam
CMD []
