FROM golang:1.23.0-alpine3.20 as builder
WORKDIR /gotemplate
RUN apk update && apk upgrade --available && sync && apk add --no-cache --virtual .build-deps
COPY . .
RUN go build -ldflags="-w -s" .
FROM alpine:3.20.2
RUN apk update && apk upgrade --available && sync
COPY --from=builder /gotemplate/gotemplate /gotemplate
ENTRYPOINT ["/gotemplate"]
