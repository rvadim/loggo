FROM golang:1.15.12 as build

COPY . /app
RUN mkdir -p /app/build

WORKDIR /app

RUN go build -o build/loggo cmd/loggo/main.go

FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=build /app/build/loggo /loggo

CMD ["/loggo"]
