# build stage
FROM golang:1.16-alpine AS builder

# gcc and musl needed by sqlite golang driver that has CGO bindings
RUN apk --no-cache add gcc musl-dev 
WORKDIR /go/src/app
COPY go.* .
RUN go mod download -x

# hack to pre build all dependencies
RUN go list -f '{{.Path}}/...' -m all | tail -n +2 | xargs -n1 go build -v -i; echo done

COPY *.go .
RUN go build -o /go/bin/app -v ./...


# final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl 
COPY --from=builder /go/bin/app /app
ENTRYPOINT  /app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 CMD curl -f localhost:8080/ping || exit 1
