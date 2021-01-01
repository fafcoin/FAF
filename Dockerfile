# Build Gfaf in a stock Go builder container
FROM golang:1.11-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-fafereum
RUN cd /go-fafereum && make gfaf

# Pull Gfaf into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-fafereum/build/bin/gfaf /usr/local/bin/

EXPOSE 8545 8546 30606 30606/udp
ENTRYPOINT ["gfaf"]
