FROM registry.access.redhat.com/hi/go:latest-fips-builder AS builder
USER 0
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=1 go build -ldflags "-w -s" -o /workspace/frontend-asset-proxy cmd/proxy/main.go

FROM registry.access.redhat.com/hi/go:latest-fips
WORKDIR /
COPY --from=builder /workspace/frontend-asset-proxy /usr/bin/frontend-asset-proxy
USER 1001
EXPOSE 8080
CMD ["/usr/bin/frontend-asset-proxy"]
