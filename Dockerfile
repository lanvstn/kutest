FROM golang:1.20-alpine AS build

COPY go.mod go.sum /kutest/
WORKDIR /kutest/
RUN go mod download

COPY . /kutest/
RUN go run github.com/onsi/ginkgo/v2/ginkgo build

###
FROM alpine:3.18

USER 1000
COPY --from=build /kutest/kutest.test /
ENTRYPOINT [ "/kutest.test", "-ginkgo.v" ]