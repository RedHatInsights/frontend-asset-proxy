FROM registry.access.redhat.com/ubi9/go-toolset:1.24.6 AS builder
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

FROM registry.access.redhat.com/ubi9-minimal:latest
WORKDIR /app
COPY --from=builder /go/bin/frontend-asset-proxy /usr/bin
ENTRYPOINT ["/usr/bin/frontend-asset-proxy"]
EXPOSE 8080
USER 1001
