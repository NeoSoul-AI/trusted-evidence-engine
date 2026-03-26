FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/tee-server ./cmd/tee-server

FROM alpine:3.21

RUN adduser -D -u 10001 tee
USER tee
WORKDIR /app

COPY --from=build /out/tee-server /app/tee-server

EXPOSE 8080

ENTRYPOINT ["/app/tee-server"]
