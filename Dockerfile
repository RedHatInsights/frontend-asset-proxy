FROM registry.access.redhat.com/ubi9/go-toolset:1.26.3-1780373831@sha256:49f5929f6674d75377902ddcc2f46baf7a5cfcaada2497ee43f66e090943afd6 AS builder
ENV GO111MODULE=on
WORKDIR $GOPATH/src/frontend-asset-proxy/
COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
COPY cmd cmd
COPY internal internal
USER root
RUN go get -v ./cmd/proxy
RUN CGO_ENABLED=0 go build -o /go/bin/frontend-asset-proxy cmd/proxy/main.go

FROM registry.access.redhat.com/ubi9-minimal:latest@sha256:ae09ecc3d754bc1726cbda3e2599cc7839e09fe1cc547ce173cf669b645be3cc
WORKDIR /app
COPY --from=builder /go/bin/frontend-asset-proxy /usr/bin
ENTRYPOINT ["/usr/bin/frontend-asset-proxy"]
EXPOSE 8080
USER 1001
