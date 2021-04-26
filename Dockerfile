FROM golang:1.16

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=10s --start-period=5s --retries=3 CMD curl -f localhost:8080/ping || exit 1

WORKDIR /go/src/app

COPY . . 

RUN go get -d -v ./...

RUN go install -v ./...

ENTRYPOINT ["go-sample"]

