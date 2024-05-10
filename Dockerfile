# First image used to build the sources
FROM golang:1.21.0 AS builder

ARG VERSION
ARG COMMIT
ARG DATE
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY . .

ENV GOPROXY=https://goproxy.io,direct
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X 'main.Version=${VERSION}}' -X 'main.Commit=${COMMIT}' -X 'main.Date=${DATE}}'" -o ./bin/silentiumd cmd/silentiumd/main.go

# Second image, running the arkd executable
FROM alpine:3.12

WORKDIR /app

COPY --from=builder /app/bin/* /app

ENV PATH="/app:${PATH}"

ENTRYPOINT [ "silentiumd" ]
    
