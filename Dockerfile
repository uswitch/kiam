FROM alpine:3.8

RUN apk --no-cache add iptables

COPY bin/kiam-linux-amd64 /kiam

CMD []
