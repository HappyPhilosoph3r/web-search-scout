# syntax=docker/dockerfile:1

FROM golang:1.21 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /web-search-scout

FROM build-stage AS test-stage

RUN go test -v ./...

FROM gcr.io/distroless/base-debian11 AS production-stage

COPY --from=build-stage /web-search-scout /web-search-scout

ENTRYPOINT [ "/web-search-scout" ]