FROM golang:1.16-stretch as build-env
WORKDIR /go/src/app
COPY . .
RUN go mod tidy
RUN go build -o fetch .

FROM debian:buster-slim
COPY --from=build-env /go/src/app/fetch /usr/local/bin
RUN apt update \
    && apt install -y --no-install-recommends ca-certificates \
    && apt -y clean \
    && rm -rf /var/lib/apt/lists/*
CMD ["fetch"]
