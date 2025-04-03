FROM golang:1.24-alpine AS build
ARG BUILD_VERSION

WORKDIR /app

COPY . .

RUN apk add --update gcc g++
RUN go mod download
RUN go build -o /mev-sp-oracle -ldflags "-X github.com/dappnode/mev-sp-oracle/config.ReleaseVersion=$BUILD_VERSION" .

FROM golang:1.24-alpine

WORKDIR /

COPY --from=build /mev-sp-oracle /mev-sp-oracle

ENTRYPOINT ["/mev-sp-oracle"]
