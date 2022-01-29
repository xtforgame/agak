FROM golang:1.12-alpine3.9

COPY agak /usr/bin
RUN chmod +x /usr/bin/agak

RUN apk add --update --no-cache ca-certificates

WORKDIR /usr/agak

# ENTRYPOINT ["/usr/bin/agak"]
CMD ["/usr/bin/agak"]
