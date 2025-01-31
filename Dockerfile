FROM golang:1.23 AS build

COPY src/ /src/
WORKDIR /src

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o powercore95 ./main.go

FROM alpine:latest
RUN apk add --no-cache \
    docker-cli

WORKDIR /app
COPY --from=build /src/powercore95 /app/powercore95
COPY --from=build /src/data   /app/data
COPY --from=build /src/templates  /app/templates
COPY --from=build /src/static  /app/static

CMD ["/app/powercore95"]
