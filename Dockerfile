FROM golang:1.16 AS build-env
WORKDIR /go/src/app
COPY . .
RUN go mod tidy
RUN go build -o fetch .


FROM gcr.io/distroless/base:latest
COPY --from=build-env /go/src/app/fetch /
CMD ["/fetch]
