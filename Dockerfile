FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /nginx-reload-q ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache nginx
COPY --from=build /nginx-reload-q /usr/local/bin/nginx-reload-q
ENTRYPOINT ["nginx-reload-q"]
