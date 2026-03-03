FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_TIME=unknown

RUN apk add --no-cache git ca-certificates

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${GIT_COMMIT} -X main.date=${BUILD_TIME}" \
    -o /keastats ./cmd/keastats

FROM alpine:3.23
COPY --from=builder /keastats /keastats
ENTRYPOINT ["cp", "/keastats", "/shared/keastats"]
