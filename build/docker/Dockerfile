#
# THORChain
#

#
# Build
#
FROM golang:1.14.7 AS build

ENV GOBIN=/go/bin
ENV GOPATH=/go
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TAG=mainnet
RUN make install

#
# Main
#
FROM alpine

RUN apk add --update jq curl bind-tools && \
    rm -rf /var/cache/apk/*

# Copy the compiled binaires over.
COPY --from=build /go/bin/generate /usr/bin/
COPY --from=build /go/bin/thord /usr/bin/
COPY --from=build /go/bin/thorcli /usr/bin/
COPY --from=build /go/bin/bifrost /usr/bin/

COPY build/scripts /scripts
