FROM alpine:3.5

RUN apk --no-cache add iptables

COPY bin/kiam /kiam

CMD ["/kiam"]