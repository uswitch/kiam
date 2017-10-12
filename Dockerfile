FROM alpine:3.5

RUN apk --no-cache add iptables

COPY bin/agent /agent
COPY bin/server /server

CMD []
