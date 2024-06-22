# syntax=docker/dockerfile:1

FROM golang:1.21.6-alpine3.19 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

RUN go build -o ./ecr-k8s-secret-manager

FROM gcr.io/distroless/static

COPY --from=build /app/ecr-k8s-secret-manager /app/ecr-k8s-secret-manager

ENTRYPOINT [ "/app/ecr-k8s-secret-manager" ]
