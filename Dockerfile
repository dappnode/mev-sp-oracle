FROM golang:1.19-alpine AS build

WORKDIR /app

COPY . .

RUN apk add --update gcc g++
RUN go mod download
RUN go build -o /mev-sp-oracle

FROM golang:1.19-alpine

WORKDIR /

COPY --from=build /mev-sp-oracle /mev-sp-oracle

ENTRYPOINT ["/mev-sp-oracle"]
