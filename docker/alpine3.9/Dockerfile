# FROM golang:1.12-alpine3.9
FROM alpine:3.9

COPY build/alpine3.9/agak /usr/bin
RUN chmod +x /usr/bin/agak

RUN apk add --update --no-cache ca-certificates

WORKDIR /usr/agak

# ENTRYPOINT ["/usr/bin/agak"]
CMD ["/usr/bin/agak"]
