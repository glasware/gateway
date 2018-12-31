FROM golang:1.11.4-alpine3.8
RUN apk --no-cache add \
    gcc\
    git\
    upx
WORKDIR /build
COPY . ./
RUN GO111MODULE=on\
    CGO_ENABLED=0\
    go build\
    -ldflags="-s -w"\
    -a\
    -installsuffix nocgo\
    -o gateway\
    *.go
RUN upx --lzma ./gateway

FROM alpine:3.8
RUN apk --no-cache add ca-certificates
RUN rm -rf /var/lib/apt/lists/*
WORKDIR /bin
COPY --from=0 /build/gateway .
COPY ./static ./static
CMD ["gateway"]