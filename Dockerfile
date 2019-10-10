FROM golang:1.13.1-alpine3.10
RUN apk --no-cache add \
    gcc\
    git\
    upx
WORKDIR /build
COPY . ./
RUN CGO_ENABLED=0\
    go build\
    -ldflags="-s -w"\
    -a\
    -installsuffix nocgo\
    -o gateway\
    *.go
RUN upx --lzma ./gateway

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
RUN rm -rf /var/lib/apt/lists/*
WORKDIR /bin
COPY --from=0 /build/gateway .
COPY ./static ./static
CMD ["gateway"]